package ccsyntax

import "errors"

func (r *rs) initLocalVariables() {
	r.ml.Lock()
	defer r.ml.Unlock()
	r.localVariable = map[string]interface{}{}
}

func (r *rs) AddLocalVariable(s string, v interface{}) error {
	r.ml.Lock()
	defer r.ml.Unlock()

	// validate duplicate entry
	if _, ok := r.localVariable[s]; ok {
		// should never happen since this is invalid YAML syntax
		// based on how we defined the local variables
		return errors.New("duplicate localVariable entry")
	}
	r.localVariable[s] = v
	return nil
}

func (r *rs) GetLocalVariable(s string) bool {
	r.ml.RLock()
	defer r.ml.RUnlock()
	_, ok := r.localVariable[s]
	return ok
}