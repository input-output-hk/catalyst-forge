package crypto

import "errors"

// StrictES256KeyManager wraps es256KeyManager to prevent KID reuse.
type StrictES256KeyManager struct {
	*es256KeyManager
}

// NewStrictES256KeyManager creates a key manager that prevents KID reuse.
func NewStrictES256KeyManager() KeyManager {
	return &StrictES256KeyManager{
		es256KeyManager: NewES256KeyManager().(*es256KeyManager),
	}
}

// AddKey adds a key to the manager, preventing KID reuse.
func (m *StrictES256KeyManager) AddKey(kid string, pemKey []byte) error {
	m.mu.RLock()
	_, exists := m.keys[kid]
	m.mu.RUnlock()
	
	if exists {
		return errors.New("key ID already exists, rotation requires new KID")
	}
	
	return m.es256KeyManager.AddKey(kid, pemKey)
}

// GenerateKey generates a new ES256 key pair, preventing KID reuse.
func (m *StrictES256KeyManager) GenerateKey(kid string) error {
	m.mu.RLock()
	_, exists := m.keys[kid]
	m.mu.RUnlock()
	
	if exists {
		return errors.New("key ID already exists, rotation requires new KID")
	}
	
	return m.es256KeyManager.GenerateKey(kid)
}