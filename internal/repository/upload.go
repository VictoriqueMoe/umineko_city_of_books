package repository

type UploadRepository interface {
	GetAllReferencedFiles() ([]string, error)
}

type uploadRepository struct {
	dao UploadRepository
}

func NewUploadRepo(dao UploadRepository) UploadRepository {
	return &uploadRepository{dao: dao}
}

func (r *uploadRepository) GetAllReferencedFiles() ([]string, error) {
	return r.dao.GetAllReferencedFiles()
}
