package rbac

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubjectType identifies the type of principal.
type SubjectType string

const (
	SubjectUser    SubjectType = "user"
	SubjectGroup   SubjectType = "group"
	SubjectService SubjectType = "service"
)

// Subject describes the principal for authorization.
type Subject struct {
	Type  SubjectType
	ID    string
	OrgID *uuid.UUID
	Attrs map[string]any
}

// ResourceRef identifies the resource being accessed.
type ResourceRef struct {
	Type   string
	ID     string
	OrgID  *uuid.UUID
	Parent *ResourceRef
	Attrs  map[string]any
}

// ResourceResolver derives a ResourceRef from the current request.
type ResourceResolver func(c *gin.Context) (ResourceRef, error)
