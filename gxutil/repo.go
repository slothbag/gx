package gxutil

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	hd "github.com/mitchellh/go-homedir"
	. "github.com/whyrusleeping/stump"
)

func (pm *PM) FetchRepo(rpath string, usecache bool) (map[string]string, error) {
	if strings.HasPrefix(rpath, "/ipns/") {
		p, err := pm.ResolveName(rpath, usecache)
		if err != nil {
			return nil, err
		}

		rpath = p
	}
	links, err := pm.Shell().List(rpath)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string)
	for _, l := range links {
		out[l.Name] = l.Hash
	}

	return out, nil
}

var ErrNotFound = errors.New("cache miss")

// TODO: once on ipfs 0.4.0, use the files api
func (pm *PM) ResolveName(name string, usecache bool) (string, error) {
	if usecache {
		cache, ok, err := CheckCacheFile(name)
		if err != nil {
			return "", err
		}

		if ok {
			return cache, nil
		}
	}

	out, err := pm.Shell().ResolvePath(name)
	if err != nil {
		Error("error from resolve path", name)
		return "", err
	}

	err = pm.cacheSet(name, out)
	if err != nil {
		return "", err
	}

	return out, nil
}

func CheckCacheFile(name string) (string, bool, error) {
	home, err := hd.Dir()
	if err != nil {
		return "", false, err
	}
	p := filepath.Join(home, ".gxcache")

	fi, err := os.Open(p)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", false, err
		}
	}
	defer fi.Close()

	cache := make(map[string]string)
	err = json.NewDecoder(fi).Decode(&cache)
	if err != nil {
		return "", false, err
	}

	v, ok := cache[name]
	if ok {
		return v, true, nil
	}
	return "", false, nil
}

// TODO: think about moving gx global files into a .config/local type thing
func (pm *PM) cacheSet(name, resolved string) error {
	home, err := hd.Dir()
	if err != nil {
		return err
	}
	p := filepath.Join(home, ".gxcache")

	_, err = os.Stat(p)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	cache := make(map[string]string)
	if err == nil { // if the file already exists
		fi, err := os.Open(p)
		if err != nil {
			return err
		}

		err = json.NewDecoder(fi).Decode(&cache)
		if err != nil {
			return err
		}

		fi.Close()
	}

	cache[name] = resolved

	fi, err := os.Create(p)
	if err != nil {
		return err
	}

	err = json.NewEncoder(fi).Encode(cache)
	if err != nil {
		return err
	}

	return fi.Close()
}

func (pm *PM) QueryRepos(query string) (map[string]string, error) {
	out := make(map[string]string)
	for name, rpath := range pm.cfg.GetRepos() {
		repo, err := pm.FetchRepo(rpath, true)
		if err != nil {
			return nil, err
		}

		if val, ok := repo[query]; ok {
			out[name] = val
		}
	}

	return out, nil
}
