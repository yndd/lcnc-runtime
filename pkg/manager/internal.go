/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	intrec "github.com/yndd/lcnc-runtime/pkg/internal/recorder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"

	//intrec "sigs.k8s.io/controller-runtime/pkg/internal/recorder"

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

const (
	// Values taken from: https://github.com/kubernetes/component-base/blob/master/config/v1alpha1/defaults.go
	defaultGracefulShutdownPeriod = 30 * time.Second
)

var _ Runnable = &controllerManager{}

type controllerManager struct {
	sync.Mutex
	started bool

	stopProcedureEngaged *int64
	errChan              chan error
	runnables            *runnables

	// cluster holds a variety of methods to interact with a cluster. Required.
	cluster cluster.Cluster

	// recorderProvider is used to generate event recorders that will be injected into Controllers
	// (and EventHandlers, Sources and Predicates).
	recorderProvider *intrec.Provider

	// controllerOptions are the global controller options.
	controllerOptions v1alpha1.ControllerConfigurationSpec

	// Logger is the logger that should be used by this manager.
	// If none is set, it defaults to log.Log global logger.
	logger logr.Logger

	// leaderElectionStopped is an internal channel used to signal the stopping procedure that the
	// LeaderElection.Run(...) function has returned and the shutdown can proceed.
	leaderElectionStopped chan struct{}

	// leaderElectionCancel is used to cancel the leader election. It is distinct from internalStopper,
	// because for safety reasons we need to os.Exit() when we lose the leader election, meaning that
	// it must be deferred until after gracefulShutdown is done.
	leaderElectionCancel context.CancelFunc

	// elected is closed when this manager becomes the leader of a group of
	// managers, either because it won a leader election or because no leader
	// election was configured.
	elected chan struct{}

	// gracefulShutdownTimeout is the duration given to runnable to stop
	// before the manager actually returns on stop.
	gracefulShutdownTimeout time.Duration

	// shutdownCtx is the context that can be used during shutdown. It will be cancelled
	// after the gracefulShutdownTimeout ended. It must not be accessed before internalStop
	// is closed because it will be nil.
	shutdownCtx context.Context

	internalCtx    context.Context
	internalCancel context.CancelFunc

	// internalProceduresStop channel is used internally to the manager when coordinating
	// the proper shutdown of servers. This channel is also used for dependency injection.
	internalProceduresStop chan struct{}
}

type hasCache interface {
	Runnable
	GetCache() cache.Cache
}

// Add sets dependencies on i, and adds it to the list of Runnables to start.
func (cm *controllerManager) Add(r Runnable) error {
	cm.Lock()
	defer cm.Unlock()
	return cm.add(r)
}

func (cm *controllerManager) add(r Runnable) error {
	// Set dependencies on the object
	if err := cm.SetFields(r); err != nil {
		return err
	}
	return cm.runnables.Add(r)
}

// Deprecated: use the equivalent Options field to set a field. This method will be removed in v0.10.
func (cm *controllerManager) SetFields(i interface{}) error {
	if err := cm.cluster.SetFields(i); err != nil {
		return err
	}
	if _, err := inject.InjectorInto(cm.SetFields, i); err != nil {
		return err
	}
	if _, err := inject.StopChannelInto(cm.internalProceduresStop, i); err != nil {
		return err
	}
	if _, err := inject.LoggerInto(cm.logger, i); err != nil {
		return err
	}
	return nil
}

func (cm *controllerManager) GetConfig() *rest.Config {
	return cm.cluster.GetConfig()
}

func (cm *controllerManager) GetClient() client.Client {
	return cm.cluster.GetClient()
}

func (cm *controllerManager) GetScheme() *runtime.Scheme {
	return cm.cluster.GetScheme()
}

func (cm *controllerManager) GetFieldIndexer() client.FieldIndexer {
	return cm.cluster.GetFieldIndexer()
}

func (cm *controllerManager) GetCache() cache.Cache {
	return cm.cluster.GetCache()
}

func (cm *controllerManager) GetEventRecorderFor(name string) record.EventRecorder {
	return cm.cluster.GetEventRecorderFor(name)
}

func (cm *controllerManager) GetRESTMapper() meta.RESTMapper {
	return cm.cluster.GetRESTMapper()
}

func (cm *controllerManager) GetAPIReader() client.Reader {
	return cm.cluster.GetAPIReader()
}

func (cm *controllerManager) GetLogger() logr.Logger {
	return cm.logger
}

func (cm *controllerManager) GetControllerOptions() v1alpha1.ControllerConfigurationSpec {
	return cm.controllerOptions
}

// Start starts the manager and waits indefinitely.
// There is only two ways to have start return:
// An error has occurred during in one of the internal operations,
// such as leader election, cache start, webhooks, and so on.
// Or, the context is cancelled.
func (cm *controllerManager) Start(ctx context.Context) (err error) {
	cm.Lock()
	if cm.started {
		cm.Unlock()
		return errors.New("manager already started")
	}
	var ready bool
	defer func() {
		// Only unlock the manager if we haven't reached
		// the internal readiness condition.
		if !ready {
			cm.Unlock()
		}
	}()

	// Initialize the internal context.
	cm.internalCtx, cm.internalCancel = context.WithCancel(ctx)

	// This chan indicates that stop is complete, in other words all runnables have returned or timeout on stop request
	stopComplete := make(chan struct{})
	defer close(stopComplete)
	// This must be deferred after closing stopComplete, otherwise we deadlock.
	defer func() {
		// https://hips.hearstapps.com/hmg-prod.s3.amazonaws.com/images/gettyimages-459889618-1533579787.jpg
		stopErr := cm.engageStopProcedure(stopComplete)
		if stopErr != nil {
			if err != nil {
				// Utilerrors.Aggregate allows to use errors.Is for all contained errors
				// whereas fmt.Errorf allows wrapping at most one error which means the
				// other one can not be found anymore.
				err = kerrors.NewAggregate([]error{err, stopErr})
			} else {
				err = stopErr
			}
		}
	}()

	// Add the cluster runnable.
	if err := cm.add(cm.cluster); err != nil {
		return fmt.Errorf("failed to add cluster to runnables: %w", err)
	}

	// Start and wait for caches.
	if err := cm.runnables.Caches.Start(cm.internalCtx); err != nil {
		if !errors.Is(err, wait.ErrWaitTimeout) {
			return err
		}
	}

	// Start the non-leaderelection Runnables after the cache has synced.
	if err := cm.runnables.Others.Start(cm.internalCtx); err != nil {
		if !errors.Is(err, wait.ErrWaitTimeout) {
			return err
		}
	}

	// Start the leader election and all required runnables.
	{
		_, cancel := context.WithCancel(context.Background())
		cm.leaderElectionCancel = cancel
		go func() {
			// Treat not having leader election enabled the same as being elected.
			if err := cm.startLeaderElectionRunnables(); err != nil {
				cm.errChan <- err
			}
			close(cm.elected)
		}()
	}

	ready = true
	cm.Unlock()
	select {
	case <-ctx.Done():
		// We are done
		return nil
	case err := <-cm.errChan:
		// Error starting or running a runnable
		return err
	}
}

// engageStopProcedure signals all runnables to stop, reads potential errors
// from the errChan and waits for them to end. It must not be called more than once.
func (cm *controllerManager) engageStopProcedure(stopComplete <-chan struct{}) error {
	if !atomic.CompareAndSwapInt64(cm.stopProcedureEngaged, 0, 1) {
		return errors.New("stop procedure already engaged")
	}

	// Populate the shutdown context, this operation MUST be done before
	// closing the internalProceduresStop channel.
	//
	// The shutdown context immediately expires if the gracefulShutdownTimeout is not set.
	var shutdownCancel context.CancelFunc
	cm.shutdownCtx, shutdownCancel = context.WithTimeout(context.Background(), cm.gracefulShutdownTimeout)
	defer shutdownCancel()

	// Start draining the errors before acquiring the lock to make sure we don't deadlock
	// if something that has the lock is blocked on trying to write into the unbuffered
	// channel after something else already wrote into it.
	var closeOnce sync.Once
	go func() {
		for {
			// Closing in the for loop is required to avoid race conditions between
			// the closure of all internal procedures and making sure to have a reader off the error channel.
			closeOnce.Do(func() {
				// Cancel the internal stop channel and wait for the procedures to stop and complete.
				close(cm.internalProceduresStop)
				cm.internalCancel()
			})
			select {
			case err, ok := <-cm.errChan:
				if ok {
					cm.logger.Error(err, "error received after stop sequence was engaged")
				}
			case <-stopComplete:
				return
			}
		}
	}()

	// We want to close this after the other runnables stop, because we don't
	// want things like leader election to try and emit events on a closed
	// channel
	defer cm.recorderProvider.Stop(cm.shutdownCtx)

	go func() {
		// First stop the non-leader election runnables.
		cm.logger.Info("Stopping and waiting for non leader election runnables")
		cm.runnables.Others.StopAndWait(cm.shutdownCtx)

		// Stop all the leader election runnables, which includes reconcilers.
		cm.logger.Info("Stopping and waiting for leader election runnables")
		cm.runnables.LeaderElection.StopAndWait(cm.shutdownCtx)

		// Stop the caches before the leader election runnables, this is an important
		// step to make sure that we don't race with the reconcilers by receiving more events
		// from the API servers and enqueueing them.
		cm.logger.Info("Stopping and waiting for caches")
		cm.runnables.Caches.StopAndWait(cm.shutdownCtx)

		// Proceed to close the manager and overall shutdown context.
		cm.logger.Info("Wait completed, proceeding to shutdown the manager")
		shutdownCancel()
	}()

	<-cm.shutdownCtx.Done()
	if err := cm.shutdownCtx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		if errors.Is(err, context.DeadlineExceeded) {
			if cm.gracefulShutdownTimeout > 0 {
				return fmt.Errorf("failed waiting for all runnables to end within grace period of %s: %w", cm.gracefulShutdownTimeout, err)
			}
			return nil
		}
		// For any other error, return the error.
		return err
	}

	return nil
}

func (cm *controllerManager) startLeaderElectionRunnables() error {
	return cm.runnables.LeaderElection.Start(cm.internalCtx)
}

func (cm *controllerManager) Elected() <-chan struct{} {
	return cm.elected
}
