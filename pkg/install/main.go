package install

import (
	"os"
)

type BasicFileStatus struct {
	Filepath string
	PermMode os.FileMode
	IsDir    bool
}
