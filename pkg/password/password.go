package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	memoryKiB   uint32 = 64 * 1024
	iterations  uint32 = 3
	parallelism uint8  = 4
	saltLen            = 16
	keyLen      uint32 = 32

	// MaxLength защищает от DoS длинными паролями (в байтах).
	MaxLength = 1024
	// Ограничиваем число одновременных хэшей: каждый занимает 64 MiB RAM.
	maxConcurrent = 4
)

var (
	ErrTooLong     = errors.New("password too long")
	ErrInvalidHash = errors.New("invalid hash format")

	b64 = base64.RawStdEncoding
	sem = make(chan struct{}, maxConcurrent)
)

func Hash(password string) (string, error) {
	if len(password) > MaxLength {
		return "", ErrTooLong
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	sem <- struct{}{}
	defer func() { <-sem }()

	key := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, keyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, iterations, parallelism, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Check проверяет пароль. Поддерживает argon2id
func Check(password, hash string) bool {
	if len(password) > MaxLength {
		return false
	}

	p, salt, key, err := decode(hash)
	if err != nil {
		return false
	}

	sem <- struct{}{}
	defer func() { <-sem }()

	got := argon2.IDKey([]byte(password), salt, p.t, p.m, p.p, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1
}

// NeedsRehash true для argon2id со старыми параметрами.
func NeedsRehash(hash string) bool {
	p, _, key, err := decode(hash)
	if err != nil {
		return true
	}
	return p.m != memoryKiB || p.t != iterations || p.p != parallelism || uint32(len(key)) != keyLen
}

var dummyHash = sync.OnceValue(func() string {
	h, _ := Hash("dummy-password")
	return h
})

// Dummy тратит столько же времени, сколько реальная проверка
// Вызываем, когда пользователь не найден, чтобы не раскрывать существование email по времени ответа
func Dummy(password string) { Check(password, dummyHash()) }

type params struct {
	m, t uint32
	p    uint8
}

func decode(enc string) (p params, salt, key []byte, err error) {
	parts := strings.Split(enc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, ErrInvalidHash
	}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.m, &p.t, &p.p); err != nil {
		return p, nil, nil, ErrInvalidHash
	}
	if salt, err = b64.DecodeString(parts[4]); err != nil {
		return p, nil, nil, ErrInvalidHash
	}
	if key, err = b64.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return p, nil, nil, ErrInvalidHash
	}
	return p, salt, key, nil
}
