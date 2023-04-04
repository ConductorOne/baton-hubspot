package hubspot

import "context"

func (t *Team) GetMembersCount(ctx context.Context) int {
	if t.UserIds == nil {
		return 0
	}

	return len(t.UserIds)
}
