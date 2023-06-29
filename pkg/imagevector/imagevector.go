// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package imagevector

import (
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/charts"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

var imageVector imagevector.ImageVector

func init() {
	var err error

	imageVector, err = imagevector.Read([]byte(charts.ImagesYAML))
	runtime.Must(err)

	imageVector, err = imagevector.WithEnvOverride(imageVector)
	runtime.Must(err)
}

// ImageVector is the image vector that contains all the needed images.
func ImageVector() imagevector.ImageVector {
	return imageVector
}

// PauseContainerImage returns the pause container image.
func PauseContainerImage() string {
	image, err := imageVector.FindImage(constants.PauseContainerImageName)
	runtime.Must(err)
	return image.String()
}

// AlpineImage returns the pause container image.
func AlpineImage() string {
	image, err := imageVector.FindImage(constants.AlpineImageName)
	runtime.Must(err)
	return image.String()
}
