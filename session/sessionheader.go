package session

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomig/cache"
	"github.com/gomig/caster"
	"github.com/gomig/utils"
)

type hSession struct {
	// cache driver
	cache cache.Cache
	// ctx request context
	ctx *fiber.Ctx
	// < 0 means 24 hours
	// > 0 is the time.Duration which the session should expire.
	expiration time.Duration
	// Session id generator
	generator func() string
	// header name
	name string
	// cache key
	key  string
	data map[string]any
}

func (ses hSession) err(
	pattern string,
	params ...any,
) error {
	return utils.TaggedError([]string{"HeaderSession"}, pattern, params...)
}

func (ses *hSession) init(
	cache cache.Cache,
	ctx *fiber.Ctx,
	exp time.Duration,
	generator func() string,
	name string,
) {
	ses.cache = cache
	ses.ctx = ctx
	ses.expiration = exp
	ses.generator = generator
	ses.name = name
	if ses.name == "" {
		ses.name = "X-SESSION-ID"
	}
	ses.data = make(map[string]any)
}

func (ses hSession) id() string {
	return "C_S_" + ses.key
}

func (ses hSession) ID() string {
	return ses.key
}

func (ses hSession) Context() *fiber.Ctx {
	return ses.ctx
}

func (ses *hSession) Parse() error {
	ses.key = ses.ctx.Get(ses.name)
	exists := false
	var err error
	if ses.key != "" {
		exists, err = ses.cache.Exists(ses.id())
		if err != nil {
			return ses.err(err.Error())
		}
	}

	if !exists {
		return ses.Regenerate()
	} else {
		res := make(map[string]any)
		caster, err := ses.cache.Cast(ses.id())
		if err != nil {
			return ses.err(err.Error())
		}

		str, err := caster.String()
		if err != nil {
			return ses.err(err.Error())
		}

		err = json.Unmarshal([]byte(str), &res)
		if err != nil {
			return ses.err(err.Error())
		}

		ses.data = res
		return nil
	}
}

func (ses *hSession) Regenerate() error {
	err := ses.Destroy()
	if err != nil {
		return err
	}

	ses.key = ses.generator()
	ses.ctx.Set(ses.name, ses.key)
	return nil
}

func (s *hSession) Set(key string, value any) {
	s.data[key] = value
}

func (ses hSession) Get(key string) any {
	return ses.data[key]
}

func (ses *hSession) Delete(key string) {
	delete(ses.data, key)
}

func (ses hSession) Exists(key string) bool {
	_, ok := ses.data[key]
	return ok
}

func (ses hSession) Cast(key string) caster.Caster {
	return caster.NewCaster(ses.data[key])
}

func (ses *hSession) Destroy() error {
	err := ses.cache.Forget(ses.id())
	if err != nil {
		return ses.err(err.Error())
	}
	ses.key = ""
	ses.data = make(map[string]any)
	return nil
}

func (ses hSession) Save() error {
	if ses.key == "" {
		return nil
	}

	data, err := json.Marshal(ses.data)
	if err != nil {
		return ses.err(err.Error())
	}

	exists, err := ses.cache.Set(ses.id(), string(data))
	if err != nil {
		return ses.err(err.Error())
	}

	if !exists {
		exp := ses.expiration
		if exp <= 0 {
			exp = 24 * time.Hour
		}

		err = ses.cache.Put(ses.id(), string(data), exp)
		if err != nil {
			return ses.err(err.Error())
		}
	}
	return nil
}
