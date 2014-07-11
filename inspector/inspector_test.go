package inspector_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/inspector"
)

var _ = Describe("Inspector", func() {
	var stackPath string
	var process ifrit.Process
	BeforeEach(func() {
		stackPath = createTmpFilePath()
		inspector.New(stackPath)
	})
	JustBeforeEach(func() {

	})
	Describe("")

	AfterEach(func() {
		deleteTmpFile()
	})
})

func createTmpFilePath() string {
	dir, err := ioutils.TempDir("", "stack_trace")
	if err != nil {
		panic(err)
	}
	return filepath.Join()
}

func deleteTmpFile(filePath string) {
	os.RemoveAll(filepath.Dir(filePath))
}
