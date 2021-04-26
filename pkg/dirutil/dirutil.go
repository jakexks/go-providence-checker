package dirutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

func CopyDirectory(srcDir, dest string) error {
	entries, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("list the content of the source directory '%s': %w", srcDir, err)
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("running syscall 'stat' on '%s': %w", sourcePath, err)
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("could not get raw syscall.Stat_t data for '%s'", sourcePath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := CreateIfNotExists(destPath, 0755); err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}
			if err := CopyDirectory(sourcePath, destPath); err != nil {
				return fmt.Errorf("copying directory: %w", err)
			}
		case os.ModeSymlink:
			if err := CopySymLink(sourcePath, destPath); err != nil {
				return fmt.Errorf("copying symlink: %w", err)
			}
		default:
			if err := Copy(sourcePath, destPath); err != nil {
				return fmt.Errorf("copying content of '%s' into '%s': %w", sourcePath, destPath, err)
			}
		}

		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return fmt.Errorf("lchown syscall on '%s': %w", destPath, err)
		}

		isSymlink := entry.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, entry.Mode()); err != nil {
				return fmt.Errorf("while copying the mode from '%s' to '%s' with chmod: %w", entry.Name(), destPath, err)
			}
		}
	}
	return nil
}

func Copy(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return fmt.Errorf("while creating the destination file: %w", err)
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	defer in.Close()
	if err != nil {
		return fmt.Errorf("while opening the source file: %w", err)
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("while copying the content of the source '%s' to the destination '%s': %w", srcFile, dstFile, err)
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
		return fmt.Errorf("readlink syscall on source symlink '%s': %w", source, err)
	}
	return os.Symlink(link, dest)
}
