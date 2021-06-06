package post_repo

import (
	"asperitas-clone/pkg/items"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	_ "go.mongodb.org/mongo-driver/mongo"
)

type PostRepo struct {
	PostDB *mgo.Collection
}

func (repo *PostRepo) GetAllPosts() ([]*items.Post, error) {
	posts := []*items.Post{}
	err := repo.PostDB.Find(bson.M{}).All(&posts)
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (repo *PostRepo) GetPostsByCategory(category string) ([]*items.Post, error) {
	posts := []*items.Post{}
	err := repo.PostDB.Find(bson.M{"category": category}).All(&posts)
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (repo *PostRepo) GetPostByID(id bson.ObjectId) (*items.Post, error) {
	post := &items.Post{}
	err := repo.PostDB.Find(bson.M{"id": id}).One(post)
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (repo *PostRepo) AddPost(post *items.Post) (bson.ObjectId, error) {
	post.ID = bson.NewObjectId()
	err := repo.PostDB.Insert(post)
	if err != nil {
		return post.ID, err
	}

	return post.ID, nil
}

func (repo *PostRepo) PostComment(post *items.Post, comment *items.Comment) (bson.ObjectId, error) {
	comment.ID = bson.NewObjectId()
	post.Comments = append(post.Comments, comment)
	err := repo.PostDB.Update(bson.M{"id": post.ID}, post)
	return comment.ID, err
}

func (repo *PostRepo) DeleteComment(post *items.Post, commentid bson.ObjectId, userid int) error {
	for ind, comment := range post.Comments {
		if comment.ID == commentid {
			if comment.Author.ID == userid {
				post.Comments = append(post.Comments[:ind], post.Comments[ind+1:]...)
				return repo.PostDB.Update(bson.M{"id": post.ID}, post)
			} else {
				return items.ErrPermissionDenied
			}
		}
	}
	return items.ErrCommentNotFound
}

func (repo *PostRepo) DeletePost(postid bson.ObjectId, user *items.User) error {
	post, err := repo.GetPostByID(postid)
	if err != nil {
		return err
	}
	if post.Author.ID == user.ID {
		err = repo.PostDB.Remove(bson.M{"id": postid})
		if err != nil {
		}
		return err
	}
	return items.ErrPermissionDenied
}

func (repo *PostRepo) DeleteUserFromVoteTry(post *items.Post, userID int) error {
	for ind, vote := range post.Votes {
		if userID == vote.User {
			post.Score -= vote.Vote
			post.Votes = append(post.Votes[:ind], post.Votes[ind+1:]...)
			return repo.PostDB.Update(bson.M{"id": post.ID}, post)
		}
	}
	return nil
}

func (repo *PostRepo) Vote(post *items.Post, userID int, vote int) error {
	if vote != 0 {
		post.Votes = append(post.Votes,
			items.Vote{
				User: userID,
				Vote: vote,
			})
	}
	post.Score += vote
	post.UpvotePercentage = 0
	for _, vote := range post.Votes {
		if vote.Vote == 1 {
			post.UpvotePercentage++
		}
	}
	if len(post.Votes) != 0 {
		post.UpvotePercentage = 100 * post.UpvotePercentage / len(post.Votes)
	} else {
		post.UpvotePercentage = 0
	}
	return repo.PostDB.Update(bson.M{"id": post.ID}, post)
}

func (repo *PostRepo) GetPostsByUsername(username string) ([]*items.Post, error) {
	posts := []*items.Post{}
	err := repo.PostDB.Find(bson.M{"author.username": username}).All(&posts)
	if err != nil {
		return nil, err
	}
	return posts, nil
}
