package session

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomig/cache"
	"github.com/gomig/caster"
	"github.com/gomig/utils"
)

// cSession cookie bases session
type cSession struct {
	// cache driver
	cache cache.Cache
	// ctx request context
	ctx *fiber.Ctx
	// secure attribute for cookie
	secure bool
	// domain attribute for cookie
	domain string
	// sameSite attribute for cookie
	// Possible values: "Lax", "Strict", "None"
	// Optional. Default: "Lax"
	sameSite string
	// < 0 means when browser closes
	// > 0 is the time.Duration which the session cookies should expire.
	expiration time.Duration
	// Session id generator
	generator func() string
	// cookie name
	name string
	// cache key
	key  string
	data map[string]any
}

func (ses cSession) err(
	pattern string,
	params ...any,
) error {
	return utils.TaggedError([]string{"CookieSession"}, pattern, params...)
}

func (ses *cSession) init(
	cache cache.Cache,
	ctx *fiber.Ctx,
	secure bool,
	domain string,
	sameSite string,
	exp time.Duration,
	generator func() string,
	name string,
) {
	ses.cache = cache
	ses.ctx = ctx
	ses.secure = secure
	ses.domain = domain
	ses.sameSite = sameSite
	ses.expiration = exp
	ses.generator = generator
	ses.name = name
	if ses.name == "" {
		ses.name = "session"
	}
	ses.data = make(map[string]any)
}

func (ses cSession) id() string {
	return "C_S_" + ses.key
}

func (ses cSession) ID() string {
	return ses.key
}

func (ses cSession) Context() *fiber.Ctx {
	return ses.ctx
}

func (ses *cSession) Parse() error {
	ses.key = ses.ctx.Cookies(ses.name)
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

func (ses *cSession) Regenerate() error {
	err := ses.Destroy()
	if err != nil {
		return err
	}

	// generate session
	ses.key = ses.generator()
	ses.data["created_at"] = time.Now().UnixNano()
	if err := ses.Save(); err != nil {
		return err
	}

	// create cookie
	cookie := fiber.Cookie{}
	cookie.Name = ses.name
	cookie.Value = ses.key
	cookie.Secure = ses.secure
	cookie.Domain = ses.domain
	cookie.SameSite = ses.sameSite
	if ses.expiration > 0 {
		cookie.Expires = time.Now().UTC().Add(ses.expiration)
	}
	ses.ctx.Cookie(&cookie)
	return nil
}

func (ses *cSession) Set(key string, value any) {
	ses.data[key] = value
}

func (ses cSession) Get(key string) any {
	return ses.data[key]
}

func (ses *cSession) Delete(key string) {
	delete(ses.data, key)
}

func (ses cSession) Exists(key string) bool {
	_, ok := ses.data[key]
	return ok
}

func (ses cSession) Cast(key string) caster.Caster {
	return caster.NewCaster(ses.data[key])
}

func (ses *cSession) Destroy() error {
	err := ses.cache.Forget(ses.id())
	if err != nil {
		return ses.err(err.Error())
	}
	ses.key = ""
	ses.data = make(map[string]any)
	return nil
}

func (ses cSession) Save() error {
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
