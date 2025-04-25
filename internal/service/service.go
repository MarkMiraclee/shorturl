package service

// ShortURLCreatorGetter определяет интерфейс для создания и получения коротких URL.
type ShortURLCreatorGetter interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
}

// PersistentStorage определяет интерфейс для хранилищ с возможностью сохранения/загрузки в файл.
type PersistentStorage interface {
	LoadFromFile(filePath string) error
	SaveToFile(filePath string) error
}

// URLStorage - композиция обоих интерфейсов (может использоваться там, где требуется обе функциональности).
type URLStorage interface {
	ShortURLCreatorGetter
	PersistentStorage
}

// URLService представляет собой реализацию сервиса сокращения URL.
type URLService struct {
	storage ShortURLCreatorGetter // Сервис зависит только от необходимого интерфейса
}

// NewURLService создает и возвращает новый экземпляр URLService.
func NewURLService(storage ShortURLCreatorGetter) *URLService {
	return &URLService{storage: storage}
}

func (s *URLService) CreateShortURL(originalURL string) (string, error) {
	return s.storage.CreateShortURL(originalURL)
}

func (s *URLService) GetOriginalURL(shortID string) (string, error) {
	return s.storage.GetOriginalURL(shortID)
}
