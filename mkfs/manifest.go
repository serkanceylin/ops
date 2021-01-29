package mkfs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

// link refers to a link filetype
type link struct {
	path string
}

// ManifestNetworkConfig has network configuration to set static IP
type ManifestNetworkConfig struct {
	IP      string
	Gateway string
	NetMask string
}

// Manifest represent the filesystem.
type Manifest struct {
	root          map[string]interface{} // root fs
	boot          map[string]interface{} // boot fs
	targetRoot    string
}

// NewManifest init
func NewManifest(targetRoot string) *Manifest {
	return &Manifest{
		root:        make(map[string]interface{}),
		boot:        make(map[string]interface{}),
		targetRoot:  targetRoot,
	}
}

// AddNetworkConfig adds network configuration
func (m *Manifest) AddNetworkConfig(networkConfig *ManifestNetworkConfig) {
	m.root["ipaddr"] = networkConfig.IP
	m.root["netmask"] = networkConfig.NetMask
	m.root["gateway"] = networkConfig.Gateway
}

// AddUserProgram adds user program
func (m *Manifest) AddUserProgram(imgpath string) {
	parts := strings.Split(imgpath, "/")
	if parts[0] == "." {
		parts = parts[1:]
	}
	program := path.Join("/", path.Join(parts...))
	err := m.AddFile(program, imgpath)
	if err != nil {
		panic(err)
	}
	m.root["program"] = program
}

// AddMount adds mount
func (m *Manifest) AddMount(label, path string) {
	MkDirPath(m.root, strings.TrimPrefix(path, "/"))
	if m.root["mounts"] == nil {
		m.root["mounts"] = map[string]string{}
	}
	mounts := m.root["mounts"].(map[string]string)
	mounts[label] = path
}

// AddEnvironmentVariable adds environment variables
func (m *Manifest) AddEnvironmentVariable(name string, value string) {
	if m.root["environment"] == nil {
		m.root["environment"] = map[string]string{}
	}
	env := m.root["environment"].(map[string]string)
	env[name] = value
}

// AddKlibs append klibs to manifest file if they don't exist
func (m *Manifest) AddKlibs(klibs []string, hostDir string) {
	klibDir := MkDir(m.boot, "klib")
	for _, klib := range klibs {
		m.AddFileTo(klibDir, klib, hostDir)
	}
	m.root["klibs"] = "bootfs"
}

// AddArgument add commandline arguments to
// user program
func (m *Manifest) AddArgument(arg string) {
	if m.root["arguments"] == nil {
		m.root["arguments"] = make([]string, 1)
	}
	args := m.root["arguments"].([]string)
	m.root["arguments"] = append(args, arg)
}

// AddDebugFlag enables debug flags
func (m *Manifest) AddDebugFlag(name string, value rune) {
	m.root[name] = value
}

// AddNoTrace enables debug flags
func (m *Manifest) AddNoTrace(name string) {
	if m.root["notrace"] == nil {
		m.root["notrace"] = make([]string, 1)
	}
	notrace := m.root["notrace"].([]string)
	m.root["notrace"] = append(notrace, name)
}

// AddKernel the kernel to use
func (m *Manifest) AddKernel(path string) {
	m.AddFileTo(m.boot, "kernel", path)
}

// AddDirectory adds all files in dir to image
func (m *Manifest) AddDirectory(dir string) error {
	err := filepath.Walk(dir, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// if the path is relative then root it to image path
		var vmpath string
		if hostpath[0] != '/' {
			vmpath = "/" + hostpath
		} else {
			vmpath = hostpath
		}

		if (info.Mode() & os.ModeSymlink) != 0 {
			info, err = os.Stat(hostpath)
			if err != nil {
				fmt.Printf("warning: %v\n", err)
				// ignore invalid symlinks
				return nil
			}

			// add link and continue on
			err = m.AddLink(vmpath, hostpath)
			if err != nil {
				return err
			}

			return nil
		}

		if info.IsDir() {
			parts := strings.FieldsFunc(vmpath, func(c rune) bool { return c == '/' })
			node := GetChildren(m.root)
			for i := 0; i < len(parts); i++ {
				if _, ok := node[parts[i]]; !ok {
					node[parts[i]] = make(map[string]interface{})
				}
				if reflect.TypeOf(node[parts[i]]).Kind() == reflect.String {
					err := fmt.Errorf("directory %s is conflicting with an existing file", hostpath)
					fmt.Println(err)
					return err
				}
				node = node[parts[i]].(map[string]interface{})
			}
		} else {
			err = m.AddFile(vmpath, hostpath)
			if err != nil {
				return err
			}
		}
		return nil

	})
	return err
}

// AddRelativeDirectory adds all files in dir to image
func (m *Manifest) AddRelativeDirectory(src string) error {
	err := filepath.Walk(src, func(hostpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		vmpath := "/" + strings.TrimPrefix(hostpath, src)

		if (info.Mode() & os.ModeSymlink) != 0 {
			info, err = os.Stat(hostpath)
			if err != nil {
				fmt.Printf("warning: %v\n", err)
				// ignore invalid symlinks
				return nil
			}

			// add link and continue on
			err = m.AddLink(vmpath, hostpath)
			if err != nil {
				return err
			}

			return nil
		}

		if info.IsDir() {
			parts := strings.FieldsFunc(vmpath, func(c rune) bool { return c == '/' })
			node := GetChildren(m.root)
			for i := 0; i < len(parts); i++ {
				if _, ok := node[parts[i]]; !ok {
					node[parts[i]] = make(map[string]interface{})
				}
				if reflect.TypeOf(node[parts[i]]).Kind() == reflect.String {
					err := fmt.Errorf("directory %s is conflicting with an existing file", hostpath)
					fmt.Println(err)
					return err
				}
				node = node[parts[i]].(map[string]interface{})
			}
		} else {
			err = m.AddFile(vmpath, hostpath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// FileExists checks if file is present at path in manifest
func (m *Manifest) FileExists(filepath string) bool {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := GetChildren(m.root)
	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			return false
		}
		node = node[parts[i]].(map[string]interface{})
	}
	pathtest := node[parts[len(parts)-1]]
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() == reflect.String {
		return true
	}
	return false
}

// AddLink to add a file to manifest
func (m *Manifest) AddLink(filepath string, hostpath string) error {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := GetChildren(m.root)

	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			node[parts[i]] = make(map[string]interface{})
		}
		node = node[parts[i]].(map[string]interface{})
	}

	pathtest := node[parts[len(parts)-1]]
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() != reflect.String {
		err := fmt.Errorf("file %s overriding an existing directory", filepath)
		fmt.Println(err)
		return err
	}

	if pathtest != nil && reflect.TypeOf(pathtest).Kind() == reflect.String && node[parts[len(parts)-1]] != hostpath {
		fmt.Printf("warning: overwriting existing file %s hostpath old: %s new: %s\n", filepath, node[parts[len(parts)-1]], hostpath)
	}

	_, err := lookupFile(m.targetRoot, hostpath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "please check your manifest for the missing file: %v\n", err)
			os.Exit(1)
		}
		return err
	}

	s, err := os.Readlink(hostpath)
	if err != nil {
		fmt.Println("bad link")
		os.Exit(1)
	}

	node[parts[len(parts)-1]] = link{path: s}
	return nil
}

// AddFile to add a file to manifest
func (m *Manifest) AddFile(filepath string, hostpath string) error {
	return m.AddFileTo(m.root, filepath, hostpath)
}

func (m *Manifest) AddFileTo(dir map[string]interface{}, filepath string, hostpath string) error {
	parts := strings.FieldsFunc(filepath, func(c rune) bool { return c == '/' })
	node := GetChildren(dir)

	for i := 0; i < len(parts)-1; i++ {
		if _, ok := node[parts[i]]; !ok {
			node[parts[i]] = make(map[string]interface{})
		}
		node = node[parts[i]].(map[string]interface{})
	}

	pathtest := node[parts[len(parts)-1]]
	if pathtest != nil && reflect.TypeOf(pathtest).Kind() != reflect.String {
		err := fmt.Errorf("file '%s' overriding an existing directory", filepath)
		fmt.Println(err)
		os.Exit(1)
	}

	if pathtest != nil && reflect.TypeOf(pathtest).Kind() == reflect.String && pathtest != hostpath {
		fmt.Printf("warning: overwriting existing file %s hostpath old: %s new: %s\n", filepath, pathtest, hostpath)
	}

	_, err := lookupFile(m.targetRoot, hostpath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "please check your manifest for the missing file: %v\n", err)
			os.Exit(1)
		}
		return err
	}

	node[parts[len(parts)-1]] = hostpath
	return nil
}

// AddLibrary to add a dependent library
func (m *Manifest) AddLibrary(path string) {
	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	parent := m.root
	for i := 0; i < len(parts)-1; i++ {
		parent = MkDir(parent, parts[i]);
	}
	m.AddFileTo(parent, parts[len(parts) - 1], path)
}

// AddUserData adds all files in dir to
// final image.
func (m *Manifest) AddUserData(dir string) {
	// TODO
}

func GetChildren(dir map[string]interface{}) map[string]interface{} {
	if dir["children"] == nil {
		dir["children"] = make(map[string]interface{})
	}
	return dir["children"].(map[string]interface{})
}

func MkDir(parent map[string]interface{}, dir string) map[string]interface{} {
	children := GetChildren(parent)
	subDir := children[dir]
	if subDir != nil {
		return subDir.(map[string]interface{})
	}
	newDir := make(map[string]interface{})
	children[dir] = newDir
	GetChildren(newDir)
	return newDir
}

func MkDirPath(parent map[string]interface{}, path string) map[string]interface{} {
	parts := strings.Split(path, "/")
	for _, element := range parts {
		parent = MkDir(parent, element)
	}
	return parent
}

func lookupFile(targetRoot string, path string) (string, error) {
	if targetRoot != "" {
		var targetPath string
		currentPath := path
		for {
			targetPath = filepath.Join(targetRoot, currentPath)
			fi, err := os.Lstat(targetPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return path, err
				}
				// lookup on host
				break
			}

			if fi.Mode()&os.ModeSymlink == 0 {
				// not a symlink found in target root
				return targetPath, nil
			}

			currentPath, err = os.Readlink(targetPath)
			if err != nil {
				return path, err
			}

			if currentPath[0] != '/' {
				// relative symlinks are ok
				path = targetPath
				break
			}

			// absolute symlinks need to be resolved again
		}
	}

	_, err := os.Stat(path)

	return path, err
}
