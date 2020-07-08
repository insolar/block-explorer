package load

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func GetJetDropsByID(a *GetJetDropByIDAttack, id string) error {
	url := fmt.Sprintf("%s%s%s", a.GetManager().GeneratorConfig.Generator.Target, "/api/v1/jet-drops/", id)
	req, _ := http.NewRequest("GET", url, nil)
	res, err := a.rc.Do(req)
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return err
}
