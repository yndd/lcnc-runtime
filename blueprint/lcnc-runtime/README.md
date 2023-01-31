# lcnc-runtime

## Description
lcnc-runtime controller

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] lcnc-runtime`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree lcnc-runtime`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init lcnc-runtime
kpt live apply lcnc-runtime --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
