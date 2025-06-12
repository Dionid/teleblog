package teleblog

import "github.com/pocketbase/pocketbase/models"

func ImagePath(
	collection *models.Collection,
	image *models.BaseModel,
	imageName string,
) string {
	return "/api/files/" + collection.Id + "/" + image.Id + "/" + imageName
}
