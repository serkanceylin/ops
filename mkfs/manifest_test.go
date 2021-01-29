package mkfs

import (
	"fmt"
	"os"
	"testing"
)

func TestAddLibs(t *testing.T) {
	m := NewManifest("")
	m.AddLibrary("/lib/x86_64-linux-gnu/libc.so.6")
	children := GetChildren(m.root)
	lib := children["lib"].(map[string]interface{})
	children = GetChildren(lib)
	fmt.Println(children)
	x86 := children["x86_64-linux-gnu"].(map[string]interface{})
	children = GetChildren(x86)
	if children["libc.so.6"] == nil {
		t.Errorf("library element not found")
	}
}

func TestManifestWithDeps(t *testing.T) {
	m := NewManifest(os.Getenv("NANOS_TARGET_ROOT"))
	m.AddUserProgram("../data/main")
	m.AddDirectory("../data/static")
}

func TestManifestWithEnv(t *testing.T) {
	m := NewManifest("")
	m.AddUserProgram("/bin/ls")
	m.AddArgument("first")
	m.AddEnvironmentVariable("var1", "value1")
	env := m.root["environment"].(map[string]string)
	if env["var1"] != "value1" {
		t.Errorf("got %v, want value1", env["var1"])
	}
}
