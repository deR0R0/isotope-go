package oauth

/*
all api routes on ion return diff values, like sex or title. this is disregarded because:
1. datatype could not be determined
2. it's literally always null....
*/

/*
sub structs in api!
*/
type Grade struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
}

type Profile struct {
	ID           int64  `json:"id"`
	Ion_Username string `json:"ion_username"`
	Display_Name string `json:"display_name"`
	Full_Name    string `json:"full_name"`
	Short_Name   string `json:"short_name"`
	First_Name   string `json:"first_name"`
	Last_Name    string `json:"last_name"`
	Nick         string `json:"nickname"`
	Email        string `json:"tj_email"`
	Grade        Grade  `json:"grade"`
	Grad_Year    int    `json:"graduation_year"`
	User_Type    string `json:"user_type"`
	Eighth_Admin bool   `json:"is_eighth_admin"`
	Teacher      bool   `json:"is_teacher"`
	Student      bool   `json:"is_student"`
}
