package detect

import "testing"

func TestPlatformHelpers(t *testing.T) {
	t.Parallel()

	fedora := Platform{OS: "linux", Distro: "fedora", IDLike: "rhel fedora"}
	if !fedora.IsFedoraLike() {
		t.Fatal("expected fedora-like platform to be detected")
	}
	if !fedora.HasNMCLI() {
		t.Fatal("expected fedora-like linux platform to report NetworkManager support")
	}

	ubuntu := Platform{OS: "linux", Distro: "ubuntu", IDLike: "debian"}
	if ubuntu.IsFedoraLike() {
		t.Fatal("did not expect ubuntu platform to be marked fedora-like")
	}
	if !ubuntu.HasNMCLI() {
		t.Fatal("expected ubuntu-like platform to report NetworkManager support")
	}

	windows := Platform{OS: "windows", Distro: "windows"}
	if windows.HasNMCLI() {
		t.Fatal("did not expect windows platform to report NetworkManager support")
	}
}
