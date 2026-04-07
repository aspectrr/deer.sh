package workflow

import (
	"errors"
	"fmt"
)

// Sentinel errors for workflow stages. These allow callers to identify
// which stage failed and take appropriate action.
var (
	// ErrDomainNotFound indicates the requested VM domain does not exist.
	ErrDomainNotFound = errors.New("domain not found")

	// ErrDomainTransient indicates the domain is transient (not persistent).
	ErrDomainTransient = errors.New("transient domains are not supported")

	// ErrDomainUnsupported indicates the domain configuration is not supported.
	ErrDomainUnsupported = errors.New("domain configuration not supported")

	// ErrSnapshotFailed indicates snapshot creation failed.
	ErrSnapshotFailed = errors.New("snapshot_failed")

	// ErrNBDAttachFailed indicates qemu-nbd attachment failed.
	ErrNBDAttachFailed = errors.New("nbd_attach_failed")

	// ErrMountFailed indicates filesystem mount failed.
	ErrMountFailed = errors.New("mount_failed")

	// ErrSanitizeFailed indicates filesystem sanitization failed.
	ErrSanitizeFailed = errors.New("sanitize_failed")

	// ErrArchiveFailed indicates rootfs archive creation failed.
	ErrArchiveFailed = errors.New("archive_failed")

	// ErrImageBuildFailed indicates Podman image build failed.
	ErrImageBuildFailed = errors.New("image_build_failed")

	// ErrContainerCreateFailed indicates container creation/start failed.
	ErrContainerCreateFailed = errors.New("container_create_failed")

	// ErrRollbackFailed indicates cleanup during rollback encountered errors.
	ErrRollbackFailed = errors.New("rollback_failed")
)

// WorkflowError wraps a stage error with additional context.
type WorkflowError struct {
	// Stage identifies which workflow step failed.
	Stage string

	// Err is the underlying sentinel or wrapped error.
	Err error

	// Detail provides additional context about the failure.
	Detail string
}

// Error implements the error interface.
func (e *WorkflowError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s: %s", e.Stage, e.Err.Error(), e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Stage, e.Err.Error())
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *WorkflowError) Unwrap() error {
	return e.Err
}

// NewWorkflowError creates a new WorkflowError.
func NewWorkflowError(stage string, err error, detail string) *WorkflowError {
	return &WorkflowError{
		Stage:  stage,
		Err:    err,
		Detail: detail,
	}
}

// ErrorResponse is the JSON structure returned on API errors.
type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}

// ToErrorResponse converts a WorkflowError to an API error response.
func (e *WorkflowError) ToErrorResponse() ErrorResponse {
	return ErrorResponse{
		Error:  e.Err.Error(),
		Detail: e.Detail,
	}
}

// Stage names for error context.
const (
	StageResolveDomain  = "resolve_domain"
	StageDetermineMode  = "determine_mode"
	StageCreateSnapshot = "create_snapshot"
	StageMountDisk      = "mount_disk"
	StageSanitizeFS     = "sanitize_filesystem"
	StageCreateArchive  = "create_archive"
	StageBuildImage     = "build_image"
	StageRunContainer   = "run_container"
	StageCleanup        = "cleanup"
)

// CleanupFunc is a function that performs cleanup and returns any error encountered.
type CleanupFunc func() error

// CleanupStack manages a stack of cleanup functions for rollback.
// Cleanups are executed in LIFO order (last registered, first executed).
type CleanupStack struct {
	funcs []CleanupFunc
}

// NewCleanupStack creates a new cleanup stack.
func NewCleanupStack() *CleanupStack {
	return &CleanupStack{
		funcs: make([]CleanupFunc, 0),
	}
}

// Push adds a cleanup function to the stack.
func (s *CleanupStack) Push(fn CleanupFunc) {
	s.funcs = append(s.funcs, fn)
}

// ExecuteAll runs all cleanup functions in reverse order.
// It collects all errors and returns a combined error if any occurred.
func (s *CleanupStack) ExecuteAll() error {
	var errs []error
	for i := len(s.funcs) - 1; i >= 0; i-- {
		if err := s.funcs[i](); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %d cleanup(s) failed: %v", ErrRollbackFailed, len(errs), errs)
	}
	return nil
}

// Clear removes all cleanup functions without executing them.
// Call this after successful completion to prevent rollback.
func (s *CleanupStack) Clear() {
	s.funcs = nil
}

// Len returns the number of registered cleanup functions.
func (s *CleanupStack) Len() int {
	return len(s.funcs)
}
