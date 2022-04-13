package converter

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func Convert(name string, data []byte) (string, []byte, error) {
	if name == "" {
		name = fmt.Sprintf("no_name_%d", time.Now().UnixMilli())
	}

	lastIdx := strings.LastIndex(name, ".")
	baseName := ""
	if lastIdx > 0 {
		baseName = name[:lastIdx]
	} else {
		baseName = name
	}

	opusName := fmt.Sprintf("%s_converted.opus", baseName)
	srcPath := fmt.Sprintf("/tmp/%s", name)
	dstPath := fmt.Sprintf("/tmp/%s", opusName)

	err := os.WriteFile(srcPath, data, 0777)
	if err != nil {
		return "", nil, fmt.Errorf("write file error: %w", err)
	}

	err = convert(srcPath, dstPath)
	if err != nil {
		return "", nil, fmt.Errorf("convert error: %w", err)
	}

	data, err = os.ReadFile(dstPath)
	if err != nil {
		return "", nil, fmt.Errorf("read file error: %w", err)
	}

	err = os.Remove(srcPath)
	if err != nil {
		return "", nil, fmt.Errorf("remove source file error: %w", err)
	}
	err = os.Remove(dstPath)
	if err != nil {
		return "", nil, fmt.Errorf("remove dest file error: %w", err)
	}

	return opusName, data, nil
}

func convert(srcPath, dstPath string) error {
	args := fmt.Sprintf(
		"-i %s -vn -c:a libopus -ac 1 -b:a 32k -vbr on -compression_level 10 -frame_duration 60 -application voip %s",
		srcPath,
		dstPath,
	)
	cmd := exec.Command("ffmpeg", strings.Split(args, " ")...)
	out, err := cmd.Output()
	fmt.Println(string(out))
	return err
}
