//go:build windows

package appdata

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func rejectNetworkPath(path string) error {
	if strings.HasPrefix(path, `\\`) {
		return errors.New("network application paths are unsupported")
	}
	volume := filepath.VolumeName(path)
	if volume != "" && windows.GetDriveType(windows.StringToUTF16Ptr(volume+`\`)) == windows.DRIVE_REMOTE {
		return errors.New("network application paths are unsupported")
	}
	return nil
}

func ReplaceFile(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return fmt.Errorf("encode source path: %w", err)
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return fmt.Errorf("encode destination path: %w", err)
	}
	if err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH); err != nil {
		return fmt.Errorf("replace %q: %w", filepath.Base(destination), err)
	}
	return nil
}

func restrictACL(path string, directory bool) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fmt.Errorf("read current Windows user: %w", err)
	}
	inheritance := uint32(0)
	if directory {
		inheritance = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	}
	accessControlList, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.ACCESS_MASK(windows.GENERIC_ALL),
		AccessMode:        windows.SET_ACCESS,
		Inheritance:       inheritance,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(user.User.Sid),
		},
	}}, nil)
	if err != nil {
		return fmt.Errorf("build private Windows ACL: %w", err)
	}
	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		accessControlList,
		nil,
	); err != nil {
		return fmt.Errorf("set private Windows ACL: %w", err)
	}
	return nil
}
