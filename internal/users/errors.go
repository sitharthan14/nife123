package users

type WrongEmailOrPasswordError struct{}

func (m *WrongEmailOrPasswordError) Error() string {
	return "wrong email or password"
}
