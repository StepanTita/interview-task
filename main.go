package tweets

import (
	...
)

type getData struct {
	authHandler dao.AuthHandler
	Client  *client.Client
	Storage dao.Storage
	Err     utils.Error
	dataSource dao.Data
}

func (impl *getPrivateTweetsImpl) Handle(params tweets.GetPrivateTweetsParams, principal interface{}) middleware.Responder {
	userId, err := impl.authHandler.ValidateBearerHeader(params.HTTPRequest.Header.Get("Authorization"))
	if err != nil {
		return tweets.TweetGetPrivateTweetsUnauthorized().WithPayload(
			&models.DefaultResponse{Message: impl.Err.InvalidBearerToken().Error()})
	}
	uid := userId.(float64)
	posts, err := impl.tweets.GetPrivateUserTweets(uint64(uid), *params.Offset, *params.Limit+1, impl.Client.DBClient)
	if err != nil {
		if err.Error() != gorm.ErrRecordNotFound.Error() {
			var isLast = true
			var previousPage, nextPage string
			return tweets.TweetGetAllTweetsOK().WithPayload(&models.TweetsList{
				Items: make([]*models.TweetsListItemsItems0, 0),
				Paging: &models.TweetsListPaging{
					IsLast:       &isLast,
					PreviousPage: &previousPage,
					NextPage:     &nextPage,
				},
			})
		}
		return tweets.TweetGetPrivateTweetsInternalServerError().WithPayload(&models.DefaultResponse{
			Message: err.Error(),
		})
	}

	for i := 0; i < len(posts); i++ {
		posts[i].ProfileImage, _ = impl.Storage.GetData(impl.Client.S3Client, posts[i].ProfileImage)
		if len(posts[i].Media) > 0 {
			var postMedia []string
			for j := 0; j < len(posts[i].Media); j++ {
				img, _ := impl.Storage.GetData(impl.Client.S3Client, posts[i].Media[j])
				postMedia = append(postMedia, img)
			}
			posts[i].Media = postMedia
		}
	}

	page := &models.Page{}
	bigint := new(big.Int)
	bigOffset, _ := bigint.SetString(posts[0].ID, 10)
	prevLink := bigint.Add(bigOffset, big.TweetInt(*params.Limit)).String()
	nextLink := posts[len(posts)-1].ID

	if *params.Offset == "" {
		nextLink = ""
	}
	if int64(len(posts)) > *params.Limit {
		page = &models.Page{
			NextPage:     nextLink,
			PreviousPage: prevLink,
			IsLast:       false,
		}
		posts = posts[:len(posts)-1]
	} else {
		page = &models.Page{NextPage: "", PreviousPage: prevLink, IsLast: true}
	}
	return tweets.TweetGetPrivateTweetsOK().WithPayload(
		&models.TweetsList{
			Paging: &models.TweetsListPaging{
				PreviousPage: &page.PreviousPage,
				NextPage:     &page.NextPage,
				IsLast:       &page.IsLast,
			},
			Items: posts,
		})
}

