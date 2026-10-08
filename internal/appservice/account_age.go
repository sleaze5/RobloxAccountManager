package appservice

import "context"

type AgeVerification string

const (
	AgeUnverified       AgeVerification = "unverified"
	AgeVerifiedFaceScan AgeVerification = "faceScan"
	AgeVerifiedID       AgeVerification = "id"
)

type accountAge struct {
	group               string
	idVerified, checked *bool
}

// An ID check also marks the age group as checked, so ID wins.
func (age accountAge) verification() AgeVerification {
	switch {
	case age.idVerified != nil && *age.idVerified:
		return AgeVerifiedID
	case age.idVerified == nil || age.checked == nil:
		return ""
	case *age.checked:
		return AgeVerifiedFaceScan
	default:
		return AgeUnverified
	}
}

func (service *Service) ageReads(ctx context.Context, accountID, version int64, age *accountAge) []accountRead {
	return []accountRead{
		{"ageGroup", func() error {
			group, err := service.users.CheckedAgeGroup(ctx, accountID, version)
			if err == nil {
				age.group, age.checked = group.Label, &group.Checked
			}
			return err
		}},
		{"ageVerification", func() error {
			verified, err := service.users.AgeVerified(ctx, accountID, version)
			if err == nil {
				age.idVerified = verified
			}
			return err
		}},
	}
}
