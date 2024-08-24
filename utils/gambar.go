package utils

import (
	"context"
	"fmt"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

var context_cloud context.Context = context.Background()
var cloudData *cloudinary.Cloudinary

func InitCloudinary() {
	cld, err := cloudinary.NewFromParams(DapatinEnvVariable("CLOUDINARY_CLOUD_NAME"), DapatinEnvVariable("CLOUDINARY_API_KEY"), DapatinEnvVariable("CLOUDINARY_API_SECRET"))

	if err != nil {
		return
	}

	cloudData = cld
}

func UploadGambar(id string) *uploader.UploadResult {
	resp, err := cloudData.Upload.Upload(context_cloud, fmt.Sprintf("./assets/gambar/%s.png", id), uploader.UploadParams{PublicID: id, Eager: "w_300,h_300"})

	if err != nil {
		fmt.Println(err)
	}

	return resp
}

func HapusGambar(id string) {
	cloudData.Upload.Destroy(context_cloud, uploader.DestroyParams{PublicID: id})
}
