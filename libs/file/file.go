package file

import (
	"archive/zip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func CopyDirectory(scrDir, dest string) error {
	entries, err := os.ReadDir(scrDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := CreateIfNotExists(destPath, 0755); err != nil {
				return err
			}
			if err := CopyDirectory(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			if err := CopySymLink(sourcePath, destPath); err != nil {
				return err
			}
		default:
			if err := Copy(sourcePath, destPath); err != nil {
				return err
			}
		}

		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}

		fInfo, err := entry.Info()
		if err != nil {
			return err
		}

		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, fInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func Copy(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func CopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}

func CreateAndCopyFromEmbed(
	folder embed.FS,
	rootFolderName string,
	filesList []fs.DirEntry,
	dest string,
) error {
	for _, file := range filesList {
		if file.IsDir() {
			content, err := folder.ReadDir(filepath.Join(rootFolderName, file.Name()))
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Join(dest, file.Name()), 0755); err != nil {
				return fmt.Errorf("failed to create directory: '%s', error: '%s'", dest, err.Error())
			}

			if err := CreateAndCopyFromEmbed(folder, filepath.Join(rootFolderName, file.Name()), content, filepath.Join(dest, file.Name())); err != nil {
				return err
			}
		} else {
			newFilePath := filepath.Join(dest, file.Name())

			if Exists(newFilePath) {
				continue
			}

			content, err := folder.ReadFile(filepath.Join(rootFolderName, file.Name()))
			if err != nil {
				return err
			}

			if err := os.WriteFile(newFilePath, content, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func CopyFromEmbed(
	folder embed.FS,
	rootFolderName string,
	dest string,
) error {
	content, err := folder.ReadDir(rootFolderName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dest, err.Error())
	}

	return CreateAndCopyFromEmbed(folder, rootFolderName, content, dest)
}

func Unzip(zipReader *zip.Reader, dist string) error {
	// Find if there's only one root folder
	var rootPath string
	rootFolders := make(map[string]bool)

	// First pass - collect all root level paths
	for _, file := range zipReader.File {
		// Skip __MACOSX folder and its contents
		if strings.HasPrefix(file.Name, "__MACOSX/") {
			continue
		}

		parts := strings.Split(strings.Trim(file.Name, "/"), "/")
		if len(parts) > 0 {
			rootFolders[parts[0]] = true
		}
	}

	// If there's exactly one root folder, use it as the prefix to trim
	if len(rootFolders) == 1 {
		for root := range rootFolders {
			rootPath = root
			break
		}
	}

	for _, file := range zipReader.File {
		// Skip __MACOSX folder and its contents
		if strings.HasPrefix(file.Name, "__MACOSX/") {
			continue
		}

		if file.FileInfo().IsDir() {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		fileBytes, err := io.ReadAll(rc)
		if err != nil {
			return err
		}

		// Remove the root directory from the path if it exists
		relativePath := file.Name
		if rootPath != "" {
			prefix := rootPath + "/"
			if strings.HasPrefix(relativePath, prefix) {
				relativePath = strings.TrimPrefix(relativePath, prefix)
			}
		}

		// Skip empty paths
		if relativePath == "" {
			continue
		}

		extractPath := filepath.Join(dist, relativePath)
		if err := os.MkdirAll(filepath.Dir(extractPath), os.ModePerm); err != nil {
			return err
		}

		if err := os.WriteFile(extractPath, fileBytes, file.Mode()); err != nil {
			return err
		}
	}

	return nil
}
