/*
 * Copyright 2023 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dgraphtest

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

func (c *LocalCluster) dgraphImage() string {
	return "dgraph/dgraph:local"
}

func (c *LocalCluster) setupBinary() error {
	isFileThere, err := fileExists(filepath.Join(binDir, fmt.Sprintf(binaryName, c.conf.version)))
	if err != nil {
		return err
	}
	if isFileThere {
		return copyBinary(binDir, c.tempBinDir, c.conf.version)
	}

	if err := ensureDgraphClone(); err != nil {
		return err
	}
	if err := runGitCheckout(c.conf.version); err != nil {
		return err
	}
	if err := buildDgraphBinary(repoDir, binDir, c.conf.version); err != nil {
		return err
	}
	return copyBinary(binDir, c.tempBinDir, c.conf.version)
}

func ensureDgraphClone() error {
	if _, err := os.Stat(repoDir); err != nil {
		return runGitClone()
	}

	if err := runGitStatus(); err != nil {
		if ierr := cleanupRepo(); ierr != nil {
			return ierr
		}
		return runGitClone()
	}

	return runGitFetch()
}

func cleanupRepo() error {
	return os.RemoveAll(repoDir)
}

func runGitClone() error {
	cmd := exec.Command("git", "clone", dgraphRepoUrl, repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "error cloning dgraph repo\noutput:%v", string(out))
	}
	return nil
}

func runGitStatus() error {
	cmd := exec.Command("git", "status")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "error running git status\noutput:%v", string(out))
	}
	return nil
}

func runGitFetch() error {
	cmd := exec.Command("git", "fetch", "-p")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "error fetching latest changes\noutput:%v", string(out))
	}
	return nil
}

func runGitCheckout(gitRef string) error {
	cmd := exec.Command("git", "checkout", "-f", gitRef)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "error checking out gitRef [%v]\noutput:%v", gitRef, string(out))
	}
	return nil
}

func buildDgraphBinary(dir, binaryDir, version string) error {
	log.Printf("[INFO] building dgraph binary")

	cmd := exec.Command("make", "dgraph")
	cmd.Dir = filepath.Join(dir, "dgraph")
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "error while building dgraph binary\noutput:%v", string(out))
	}
	if err := copy(filepath.Join(dir, "dgraph", "dgraph"),
		filepath.Join(binaryDir, fmt.Sprintf(binaryName, version))); err != nil {
		return errors.Wrap(err, "error while copying binary")
	}
	return nil
}

func copyBinary(fromDir, toDir, version string) error {
	if err := copy(filepath.Join(fromDir, fmt.Sprintf(binaryName, version)),
		filepath.Join(toDir, "dgraph")); err != nil {
		return errors.Wrap(err, "error while copying binary into tempBinDir")
	}
	return nil
}

func copy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return errors.Wrap(err, fmt.Sprintf("%s is not a regular file", src))
	}

	source, err := os.Open(src)
	srcStat, _ := source.Stat()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error while opening file [%s]", src))
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if err := os.Chmod(dst, srcStat.Mode()); err != nil {
		return err
	}
	_, err = io.Copy(destination, source)
	return err
}
