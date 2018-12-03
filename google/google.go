package google

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/admin/directory/v1"
	goauth "google.golang.org/api/oauth2/v2"
)

type GoogleProvider struct {
	serviceAccount   []byte
	ImpersonateAdmin string
}

type Group struct {
	group    *admin.Group
	provider *GoogleProvider
	Members  []string
}

func (g *Group) String() string {
	groupAliases := strings.Join(g.group.Aliases, ", ")
	if len(groupAliases) > 0 {
		groupAliases = fmt.Sprintf(" [%s]", groupAliases)
	}
	return fmt.Sprintf("%s (%s)%s\n", g.group.Name, g.group.Email, groupAliases)
}

func (p *GoogleProvider) oauth2Exchange(ctx context.Context, code string, config *oauth2.Config) (*oauth2.Token, error) {
	return config.Exchange(ctx, code)
}

func (p *GoogleProvider) authUser(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (*goauth.Userinfoplus, error) {
	client := config.Client(ctx, token)
	userService, err := goauth.New(client)
	if err != nil {
		return nil, err
	}

	user, err := goauth.NewUserinfoV2MeService(userService).Get().Do()
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (p *GoogleProvider) directoryService(ctx context.Context) (*admin.Service, error) {
	jwtConfig, err := google.JWTConfigFromJSON(p.serviceAccount, admin.AdminDirectoryUserScope, admin.AdminDirectoryGroupScope)
	if err != nil {
		return nil, fmt.Errorf("unable to create JWT from service account: %v", err)
	}
	jwtConfig.Subject = p.ImpersonateAdmin

	client := jwtConfig.Client(ctx)

	srv, err := admin.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to create directory service: %v", err)
	}
	return srv, nil
}

func (p *GoogleProvider) ListGroups(ctx context.Context) (groups []*Group, err error) {
	return p.groupsPerUser(ctx, "")
}

func (p *GoogleProvider) usersPerGroup(ctx context.Context, groupKey string) (members []string, err error) {
	svc, err := p.directoryService(ctx)
	if err != nil {
		return []string{}, err
	}

	query := svc.Members.List(groupKey)

	for {
		resp, err := query.Do()
		if err != nil {
			return []string{}, err
		}
		for _, m := range resp.Members {
			members = append(members, m.Email)
		}

		if resp.NextPageToken == "" {
			break
		}
		query.PageToken(resp.NextPageToken)
	}

	return members, nil
}

func (p *GoogleProvider) groupsPerUser(ctx context.Context, userKey string) (groups []*Group, err error) {
	svc, err := p.directoryService(ctx)
	if err != nil {
		return []*Group{}, err
	}

	query := svc.Groups.List().UserKey(userKey)

	for {
		resp, err := query.Do()
		if err != nil {
			return []*Group{}, err
		}
		for _, g := range resp.Groups {
			members, err := p.usersPerGroup(ctx, g.Id)
			if err != nil {
				return []*Group{}, err
			}
			groupElem := &Group{group: g, provider: p, Members: members}
			groups = append(groups, groupElem)
		}

		if resp.NextPageToken == "" {
			break
		}
		query.PageToken(resp.NextPageToken)
	}

	return groups, nil
}

func New(serviceAccountPath string) (*GoogleProvider, error) {
	serviceAccountBytes, err := ioutil.ReadFile(serviceAccountPath)
	if err != nil {
		return nil, fmt.Errorf("error reading service account file: %v", err)
	}

	return &GoogleProvider{
		serviceAccount: serviceAccountBytes,
	}, nil

}
