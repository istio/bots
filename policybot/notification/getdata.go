// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package notification

import (
	"context"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/analytics/v3"

	"istio.io/bots/policybot/pkg/cmdutil"
	"istio.io/bots/policybot/pkg/config"
	"istio.io/pkg/log"
)

const (
	datelayout string = "2006-01-02"
)

var (
	gaServiceAcctEmail  string = "istioioanalytics@istio-testing.iam.gserviceaccount.com"
	gaServiceAcctPEMKey string = "./analyticsdumper.pem"
	gaTableID           string = "ga:149552698"
)

var (
	enddate   string = time.Now().Format(datelayout)
	startdate string = time.Now().Add(time.Hour * 24 * -1).Format(datelayout)
	metric    string = "ga:uniquePageviews"
	tokenurl  string = "https://accounts.google.com/o/oauth2/token"
)

func findEmailToWhom(addr string, docsOwnerMap map[string]string) string {
	for page, owner := range docsOwnerMap {
		if strings.Contains(addr, page) {
			return owner
		}
	}
	return "istio/wg-docs-maintainers"
}
func findEmailToWhomMap(ownersMap map[string]map[string]string, addrs []string,
	event string) (map[string]map[string]string, error) {
	docsOwnerMap, err := ReadDocsOwner()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		owner := findEmailToWhom(addr, docsOwnerMap)
		_, ok := ownersMap[owner]
		if !ok {
			ownersMap[owner] = make(map[string]string)
		}
		ownersMap[owner][event] += addr + "<br>"
	}
	return ownersMap, nil
}

func getData(dataGaGetCall *analytics.DataGaGetCall, dimensions string, filter string, sort string) *analytics.GaData {
	//set up dimension
	if dimensions != "" {
		dataGaGetCall.Dimensions(dimensions)
	}
	// setup the filter
	if filter != "" {
		dataGaGetCall.Filters(filter)
	}
	//set up sort
	if sort != "" {
		dataGaGetCall.Sort(sort)
	}
	getData, err := dataGaGetCall.Do()
	if err != nil {
		log.Errorf("can't achieve data using analytics API: %v", err)
	}
	return getData
}

func getInternal404Error(dataService *analytics.DataGaService) *analytics.GaData {
	dataGaGetCall := dataService.Get(gaTableID, startdate, enddate, metric)
	filter := "ga:previousPagePath!=(entrance);ga:pageTitle=~404" //https://developers.google.com/analytics/devguides/reporting/core/v3/reference#filters
	dimensions := "ga:pageTitle,ga:pagePath,ga:previousPagePath"
	sort := "-ga:uniquePageviews"
	return getData(dataGaGetCall, dimensions, filter, sort)
}

func getExternal404Error(dataService *analytics.DataGaService) *analytics.GaData {
	dataGaGetCall := dataService.Get(gaTableID, startdate, enddate, metric)
	filter := "ga:previousPagePath==(entrance);ga:pageTitle=~404"
	dimensions := "ga:pageTitle,ga:pagePath,ga:previousPagePath"
	sort := "-ga:uniquePageviews"
	return getData(dataGaGetCall, dimensions, filter, sort)
}

func getRepeatedBadReviews(dataService *analytics.DataGaService) *analytics.GaData {
	metrics := "ga:totalEvents,ga:eventValue"
	dataGaGetCall := dataService.Get(gaTableID, startdate, enddate, metrics)
	dimensions := "ga:eventCategory,ga:pagePath"
	filter := "ga:totalEvents>10"
	return getData(dataGaGetCall, dimensions, filter, "")
}

func pageviewIncrease(dataService *analytics.DataGaService) *analytics.GaData {
	metrics := "ga:pageviews"
	dataGaGetCall := dataService.Get(gaTableID, startdate, enddate, metrics)
	dimensions := "ga:pageTitle"
	return getData(dataGaGetCall, dimensions, "", "")
}

func getDataService() (dataService *analytics.DataGaService) {
	/*
		key, err := ioutil.ReadFile(gaServiceAcctPEMKey)

		if err != nil {
			log.Errorf("error reading service account private key: %v", err)
		}

		jwtConf := jwt.Config{
			Email:      gaServiceAcctEmail,
			PrivateKey: key,
			Scopes:     []string{analytics.AnalyticsReadonlyScope},
			TokenURL:   tokenurl,
		}

		client := jwtConf.Client(context.Background()) */
	ctx := context.Background()
	analyticService, err := analytics.NewService(ctx)

	if err != nil {
		log.Errorf("error creating analytics service: %v", err)
	}

	dataService = analytics.NewDataGaService(analyticService)
	return
}

func getLink(getData *analytics.GaData) (linkList []string) {
	for row := 0; row <= len(getData.Rows)-1; row++ {
		linkList = append(linkList, getData.Rows[row][0])
	}
	return
}

func HourlyReport(reg *config.Registry, secrets *cmdutil.Secrets) error {
	message := ""
	sendMessage := false
	//check if website istio.io is down
	resp, err := http.Get("https://istio.io")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		sendMessage = true
		message += "istio.io is down:" + err.Error()		
	}
	defer resp.Body.Close()
	//check if website preliminary.istio.io is down
	resp, err = http.Get("https://preliminary.istio.io/")
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		sendMessage = true
		message += "preliminary.istio.io is down:" + err.Error()		
	}
	defer resp.Body.Close()

	if sendMessage {
		err = Send("Website is down", "", message, reg, secrets)
		return err
	}
	return nil
}

func DailyReport(reg *config.Registry, secrets *cmdutil.Secrets) error {
	dataService := getDataService()
	internal404Message := getInternal404Error(dataService)
	external404Message := getExternal404Error(dataService)

	OwnersMap := make(map[string]map[string]string)
	OwnersMap, err := findEmailToWhomMap(OwnersMap, getLink(internal404Message), "internal404error")
	if err != nil {
		return err
	}
	OwnersMap, err = findEmailToWhomMap(OwnersMap, getLink(external404Message), "external404error")
	if err != nil {
		return err
	}

	for owner, eventsMap := range OwnersMap {
		message := ""
		for event, link := range eventsMap {
			message += event + link
		}
		err := Send("404 Error", owner+" : ", message, reg, secrets)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
func Test(reg *config.Registry, secrets *cmdutil.Secrets){
	var data analytics.GaData
	rows := [][]string{}
	rows = append(rows, []string{"/docs/reference/config/installation-options/index.html", "Istio / 404 Page not found", "(entrance)", "24"})
	rows = append(rows, []string{"/docs/reference/config/istio.authentication.v1alpha1/index.html", "Istio / 404 Page not found", "(entrance)", "2"})
	rows = append(rows, []string{"/docs/reference/config/istio.networking.v1alpha3/index.html", "Istio / 404 Page not found", "(entrance)", "34"})
	data.Rows = rows
	internal404Message := &data

	OwnersMap := make(map[string]map[string]string)
	OwnersMap = findEmailToWhomMap(OwnersMap, getLink(internal404Message), "Internal 404 error")

	for owner, eventsMap := range OwnersMap {
		message := ""
		for event, link := range eventsMap {
			message += event + ": <br>" + link
		}
		Send("404 Error", owner+" : ", message, reg, secrets)
	}

} */
