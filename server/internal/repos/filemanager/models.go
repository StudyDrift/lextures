package filemanager

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID        uuid.UUID  `json:"id"`
	CourseID  uuid.UUID  `json:"courseId"`
	ParentID  *uuid.UUID `json:"parentId"`
	Name      string     `json:"name"`
	CreatedBy *uuid.UUID `json:"createdBy"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type FileItem struct {
	ID               uuid.UUID  `json:"id"`
	CourseID         uuid.UUID  `json:"courseId"`
	FolderID         *uuid.UUID `json:"folderId"`
	StorageKey       string     `json:"storageKey"`
	OriginalFilename string     `json:"originalFilename"`
	DisplayName      string     `json:"displayName"`
	MimeType         string     `json:"mimeType"`
	ByteSize         int64      `json:"byteSize"`
	UploadedBy       *uuid.UUID `json:"uploadedBy"`
	CanvasFileID     *int64     `json:"canvasFileId,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// FolderBreadcrumb is one segment of the path from the course-files root to the current folder.
type FolderBreadcrumb struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// FolderContents is the response for listing a folder (or the root).
type FolderContents struct {
	FolderID    *uuid.UUID         `json:"folderId"`
	Breadcrumbs []FolderBreadcrumb `json:"breadcrumbs,omitempty"`
	Folders     []Folder           `json:"folders"`
	Files       []FileItem         `json:"files"`
}
