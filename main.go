package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver"
	"github.com/Masterminds/vcs"
	"github.com/pelletier/go-toml"
)

const ManifestLockFile = "Gopkg.lock"
const ManifestFile = "Gopkg.toml"

type rawManifest struct {
	Constraints []rawProject `toml:"constraint,omitempty"`
	Overrides   []rawProject `toml:"override,omitempty"`
	Ignored     []string     `toml:"ignored,omitempty"`
	Required    []string     `toml:"required,omitempty"`
}
type rawLock struct {
	Projects []rawProject `toml:"projects,omitempty"`
}
type rawProject struct {
	Name     string `toml:"name"`
	Branch   string `toml:"branch,omitempty"`
	Revision string `toml:"revision,omitempty"`
	Version  string `toml:"version,omitempty"`
	Source   string `toml:"source,omitempty"`
}
type update struct {
	Name, Current, New string
	Updatable          bool
}

func main() {
	var err error
	lockProjects := &rawLock{}
	if _, err = os.Stat(ManifestLockFile); os.IsNotExist(err) {
		panic("lock file does not exists, can't work with it yet")
	}
	b, err := ioutil.ReadFile(ManifestLockFile)
	if err != nil {
		panic(err)
	}
	err = toml.Unmarshal(b, lockProjects)
	if err != nil {
		panic(fmt.Sprintf("unable to load TomlTree from string: %s", err))
	}
	manifest := &rawManifest{}
	if _, err = os.Stat(ManifestFile); os.IsNotExist(err) {
		panic(fmt.Sprintf("file %s does not exists", ManifestLockFile))
	}
	b, err = ioutil.ReadFile(ManifestFile)
	if err != nil {
		panic(err)
	}
	err = toml.Unmarshal(b, manifest)
	if err != nil {
		panic(fmt.Sprintf("unable to load TomlTree from string: %s", err))
	}
	// fmt.Printf("%#v\n\n", manifest)
	tomlProjects := make(map[string]rawProject, len(manifest.Constraints))
	for _, v := range manifest.Constraints {
		if v.Version != "" {
			tomlProjects[v.Name] = v
		}
	}
	for _, v := range lockProjects.Projects {
		if _, ok := tomlProjects[v.Name]; ok {
			tomlProjects[v.Name] = v
		}
	}
	// fmt.Printf("%#v\n", tomlProjects)

	ch := make(chan update)
	go func() {
		fmt.Println("Current\t\t New\t\t Name")
		var i int
		fmt.Printf("%d/%d", i, len(tomlProjects))
		for v := range ch {
			i++
			fmt.Print("\r")
			if v.Updatable {
				fmt.Printf("%s\t\t %s\t\t %s\n", v.Current, v.New, v.Name)
			}
			fmt.Printf("%d/%d", i, len(tomlProjects))
		}
	}()
	wg := sync.WaitGroup{}
	wg.Add(len(tomlProjects))
	for _, v := range tomlProjects {
		go func(v rawProject) {
			tags, err := getTags(v)
			if err != nil {
				panic(err)
			}
			var lastVer *semver.Version
			if len(tags) > 0 {
				lastVer = tags[len(tags)-1]
			}
			lockVer, _ := semver.NewVersion(v.Version)

			updatable := lockVer != nil && lastVer != nil && lastVer.GreaterThan(lockVer)
			ch <- update{v.Name, v.Version, lastVer.String(), updatable}

			wg.Done()
		}(v)
	}
	wg.Wait()
	close(ch)
}

func getTags(v rawProject) (versions, error) {
	remote := v.Source
	httpsRemote := ""
	if remote == "" {
		httpsRemote = "https://" + v.Name
		parsed, err := url.Parse(httpsRemote)
		if err != nil {
			return nil, fmt.Errorf("parsing error: %s", err)
		}
		remote = "git@" + parsed.Host + ":" + parsed.Path[1:] + ".git"
	}
	local, _ := ioutil.TempDir("", "go-vcs")
	defer os.RemoveAll(local)
	repo, err := getRepo(local, remote, httpsRemote)
	if err != nil {
		return nil, fmt.Errorf("unable to get repo: %s", err)
	}
	tags, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("unable to get tags: %s", err)
	}
	var vs versions
	for _, tag := range tags {
		if ver, err := semver.NewVersion(tag); err == nil {
			vs = append(vs, ver)
		}
	}
	sort.Sort(vs)
	return vs, nil
}

func getRepo(tmpFolder, sshRemote, httpsFallback string) (vcs.Repo, error) {
	repo, err := vcs.NewRepo(sshRemote, tmpFolder)
	if err != nil {
		return nil, fmt.Errorf("unable to init vcs: %s", err)
	}
	err = repo.Get()
	if err != nil {
		if strings.Contains(err.Error(), "Operation timed out") && httpsFallback != "" {
			// fmt.Println("retrying with https", httpsFallback)
			repo, err = vcs.NewRepo(httpsFallback, tmpFolder)
			if err != nil {
				return nil, fmt.Errorf("unable to init vcs https: %s", err)
			}
			err = repo.Get()
		}
	}
	if err != nil {
		return nil, fmt.Errorf("unable to get vcs: %s", err)
	}
	return repo, nil
}

type versions []*semver.Version

func (s versions) Len() int {
	return len(s)
}
func (s versions) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s versions) Less(i, j int) bool {
	return s[i].LessThan(s[j])
}
