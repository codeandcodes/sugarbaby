package shared

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	pb "github.com/codeandcodes/subs/protos"
)

type UserService struct {
	pb.UnimplementedUserServiceServer
	FsClient *firestore.Client
}

type FsUser struct {
	EmailAddress      string
	FbUserId          string
	OsUserId          string
	SquareAccessToken string
	DisplayName       string
	PhotoUrl          string
}

type UserNotFoundError string
type FirestoreError string

func (e UserNotFoundError) Error() string {
	return fmt.Sprintf("User not found in firestore! %v", string(e))
}

func (e FirestoreError) Error() string {
	return fmt.Sprintf("Something went wrong with firestore %v", string(e))
}

func (s *UserService) RegisterUser(ctx context.Context, in *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	log.Printf("Registering user: %v", in.EmailAddress)

	doc, _, err := s.FsClient.Collection("users").Add(context.Background(), map[string]interface{}{
		"FbUserId":     in.FbUserId,
		"EmailAddress": in.EmailAddress,
		"DisplayName":  in.DisplayName,
		"PhotoUrl":     in.PhotoUrl,
	})
	if err != nil {
		log.Printf("Failed adding user to firestore: %v", err)
		return nil, err
	}

	fsUser, err := s.GetUser(ctx, doc.ID)
	if err != nil {
		return nil, err
	}

	return &pb.RegisterUserResponse{
		OsUserId:     doc.ID,
		EmailAddress: fsUser.EmailAddress,
		FbUserId:     fsUser.FbUserId,
		HttpResponse: &pb.HttpResponse{
			Message:    "Successfully registered user.",
			StatusCode: fmt.Sprintf("%d", http.StatusOK),
		},
	}, nil
}

func (s *UserService) AddSquareAccessToken(ctx context.Context, in *pb.AddSquareAccessTokenRequest) (*pb.AddSquareAccessTokenResponse, error) {
	log.Printf("Calling AddSquareAccessToken as %v", ctx.Value("UserId"))

	// TODO: get this from the context instead of directly from the request once auth is in place
	osUserId := fmt.Sprintf("%v", ctx.Value("UserId"))

	doc := s.FsClient.Collection("users").Doc(osUserId)

	_, err := doc.Set(ctx, map[string]interface{}{
		"SquareAccessToken": in.SquareAccessToken,
	}, firestore.MergeAll)

	if err != nil {
		// Handle any errors in an appropriate way, such as returning them.
		log.Printf("An error has occurred storing square access token: %s", err)
		return nil, err
	}

	return &pb.AddSquareAccessTokenResponse{
		HttpResponse: &pb.HttpResponse{
			Message:    "Successfully associated token.",
			StatusCode: fmt.Sprintf("%d", http.StatusOK),
		},
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, userId string) (*FsUser, error) {

	dsnap, err := s.FsClient.Collection("users").Doc(userId).Get(ctx)
	if err != nil {
		return nil, FirestoreError(fmt.Sprintf("%v", err))
	}

	if !dsnap.Exists() {
		return nil, UserNotFoundError(userId)
	}

	var fsUser FsUser
	dsnap.DataTo(&fsUser)
	return &fsUser, nil
}