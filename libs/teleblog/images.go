package teleblog

import "github.com/pocketbase/pocketbase/models"

func ImagePath(
	collection *models.Collection,
	image *models.BaseModel,
	imageName string,
) string {
	if imageName == "" {
		return ""
	}

	return "/api/files/" + collection.Id + "/" + image.Id + "/" + imageName
}
