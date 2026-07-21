package repository

type UploadRepository interface {
	GetAllReferencedFiles() ([]string, error)
}
