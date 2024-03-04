package ocitool

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type mergeCmd struct {
	Layouts []string
	Out     string
}

func newMergeCommand() *cobra.Command {
	mc := &mergeCmd{}

	cmd := &cobra.Command{
		Use:  "merge",
		RunE: mc.Run,
	}

	f := cmd.Flags()
	f.StringSliceVar(&mc.Layouts, "layout", []string{}, "layouts to merge")
	f.StringVar(&mc.Out, "out", "_output", "output directory")

	if err := cmd.MarkFlagRequired("layout"); err != nil {
		log.Fatalf("unable to set layout flag required: %s", err)
	}

	return cmd
}

func (m *mergeCmd) Run(_ *cobra.Command, _ []string) error {
	r := regexp.MustCompile("(.+)-(linux|windows)-(arm64|amd64)")
	layouts := make(map[string][]string)
	for _, layout := range m.Layouts {
		dir := filepath.Base(layout)
		m := r.FindStringSubmatch(dir)
		if len(m) < 1 {
			log.Fatalf("invalid layout path: %s", layout)
		}
		image := m[1]
		fmt.Printf("layout: %s", image)
		layouts[image] = append(layouts[image], layout)
	}
	for image, images := range layouts {
		log.Infof("image: %s", image)
		log.Infof("images: %s", images)
		log.Infof("creating: %s", filepath.Join(m.Out, image))
		os.MkdirAll(filepath.Join(m.Out, image), 0755)
		log.Infof("creating: %s", filepath.Join(m.Out, image, "blobs", "sha256"))
		os.MkdirAll(filepath.Join(m.Out, image, "blobs", "sha256"), 0755)
		index := &ocispec.Index{
			Versioned: specs.Versioned{
				SchemaVersion: 2,
			},
			MediaType: "application/vnd.oci.image.index.v1+json",
		}

		for _, i := range images {
			idx, err := ReadIndex(i)
			if err != nil {
				log.Fatalf("unable to read index for %s: %s", i, err)
			}
			index.Manifests = append(index.Manifests, idx.Manifests[0])

			log.Infof("image: %s", i)
			if err := CopyBlobs(
				filepath.Join(i, "blobs", "sha256"),
				filepath.Join(m.Out, image, "blobs", "sha256"),
			); err != nil {
				log.Fatalf("error copying: %s", err)
			}
		}

		b, err := json.Marshal(index)
		if err != nil {
			log.Fatalf("error writing index: %s", err)
		}
		os.WriteFile(filepath.Join(m.Out, image, "index.json"), b, fs.ModePerm)
	}
	return nil
}

func ReadIndex(image string) (*ocispec.Index, error) {
	var index ocispec.Index
	b, err := os.ReadFile(filepath.Join(image, "index.json"))
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &index)
	if err != nil {
		return nil, err
	}
	return &index, nil
}

func CopyBlobs(src, dst string) error {
	log.Infof("reading: %s", src)
	files, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("invalid source dir: %s", err)
	}
	for _, file := range files {
		log.Infof("file: %s", file.Name())
		f, err := os.Open(filepath.Join(src, file.Name()))
		if err != nil {
			return err
		}
		defer f.Close()
		log.Infof("creating: %s\n", filepath.Join(dst, file.Name()))
		out, err := os.Create(filepath.Join(dst, file.Name()))
		if err != nil {
			return err
		}
		defer func() {
			if e := out.Close(); e != nil {
				err = e
			}
		}()
		_, err = io.Copy(f, out)
		if err != nil {
			return err
		}

		err = out.Sync()
		if err != nil {
			return err
		}

		si, err := os.Stat(filepath.Join(src, file.Name()))
		if err != nil {
			return err
		}
		err = os.Chmod(filepath.Join(dst, file.Name()), si.Mode())
		if err != nil {
			return err
		}
	}
	return nil
}
