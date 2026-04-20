package application

import (
	"context"

	"github.com/behzod/pageSDK/internal/domain"
	"github.com/behzod/pageSDK/internal/ports"
	"github.com/behzod/pageSDK/internal/runtime"
)

// PageService — сервис загрузки страниц с инициализацией state.
type PageService struct {
	pageRegistry ports.PageRegistry
	stateManager *StateManager
	pageLoader   *runtime.PageLoader
}

func NewPageService(reg ports.PageRegistry, sm *StateManager, loader *runtime.PageLoader) *PageService {
	return &PageService{
		pageRegistry: reg,
		stateManager: sm,
		pageLoader:   loader,
	}
}

// LoadPage возвращает PageInstance для сессии. Если state нет — создаётся новый.
func (s *PageService) LoadPage(ctx context.Context, pageID, sessionID string) (*domain.PageInstance, error) {
	form, err := s.pageRegistry.Get(pageID)
	if err != nil {
		return nil, err
	}

	state, err := s.stateManager.GetOrCreate(ctx, sessionID, pageID)
	if err != nil {
		return nil, err
	}

	page := s.pageLoader.Load(pageID, form, state)
	return page, nil
}
