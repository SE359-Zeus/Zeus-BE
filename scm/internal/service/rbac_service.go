package service

import (
	"fmt"
	"sync"
)

type EndpointRole struct {
	Method        string
	Path          string
	RequiredLevel string
}

var levelRank = map[string]int{
	"Administrator": 3,
	"Operator":      2,
	"Worker":        1,
}

type RBACService struct {
	mu             sync.RWMutex
	endpointLevels map[string]string
	roleLevels     map[string]int
}

func NewRBACService(endpoints []EndpointRole, roleLevels map[string]int) *RBACService {
	s := &RBACService{
		endpointLevels: make(map[string]string, len(endpoints)),
		roleLevels:     roleLevels,
	}
	for _, e := range endpoints {
		key := fmt.Sprintf("%s:%s", e.Method, e.Path)
		s.endpointLevels[key] = e.RequiredLevel
	}
	return s
}

func (s *RBACService) GetRequiredLevel(method, path string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", method, path)
	level, ok := s.endpointLevels[key]
	if !ok {
		return "", nil
	}
	return level, nil
}

func (s *RBACService) getRoleLevel(roleName string) int {
	if level, ok := s.roleLevels[roleName]; ok {
		return level
	}
	if level, ok := levelRank[roleName]; ok {
		return level
	}
	return 1
}

func (s *RBACService) CanAccess(roleName, method, path string) (bool, error) {
	requiredLevel, err := s.GetRequiredLevel(method, path)
	if err != nil {
		return false, err
	}
	if requiredLevel == "" {
		return true, nil
	}

	roleRank := s.getRoleLevel(roleName)
	reqRank := s.getRoleLevel(requiredLevel)
	return roleRank >= reqRank, nil
}
