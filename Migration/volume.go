package Migration
import (
"github.com/docker/docker/api/types/volume"
"github.com/docker/docker/client"
"context"
"fmt"
"io"
"os"
"path/filepath"
"archive/tar"
"bytes"
"compress/gzip"
pb "github.com/zhiyuanGH/container-joint-migration/pkg/migration"
)

func Createvolume(volres *pb.VolumeResponse) (binds string, err error) {
	if volres.NfsSource != "" {
		return createVolumeFromNFS( volres.NfsSource, volres.Destination)
	}
	return createVolumeFromData(volres.VolumeName, volres.VolumeData,volres.Destination)
}

func createVolumeFromNFS(NfsSource string, destination string) (binds string, err error){
	return fmt.Sprintf("%s:/%s",NfsSource,destination),nil
}


func createVolumeFromData(volumeName string, volumeData []byte, destination string) (binds string, err error ){
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "",err
	}

	_, err = cli.VolumeCreate(context.Background(), volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return "",err
	}

	volumeDir := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", volumeName)
	os.MkdirAll(volumeDir, os.ModePerm)

	buf := bytes.NewBuffer(volumeData)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return "", err
	}
	tarReader := tar.NewReader(gz)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "",err
		}

		target := filepath.Join(volumeDir, hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return "", err
			}
		} else {
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return"", err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				return "",err
			}
			f.Close()
		}
	}

	return fmt.Sprintf("%s:/%s",volumeName,destination),nil
}
