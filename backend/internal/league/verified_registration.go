package league

import "context"

type verifiedPlayerEmailKey struct{}

// WithVerifiedPlayerEmail is for registration handlers after account authentication.
func WithVerifiedPlayerEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, verifiedPlayerEmailKey{}, email)
}

func VerifiedPlayerEmail(ctx context.Context) string {
	email, _ := ctx.Value(verifiedPlayerEmailKey{}).(string)
	return email
}
