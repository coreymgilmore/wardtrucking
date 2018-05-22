/*Package ward provides tooling to connect to the Ward Trucking API.  This is for truck shipments,
not small parcels.  Think LTL (less than truckload) shipments.  This code was created off the Ward API
documentation.  This uses and XML SOAP API.

You will need to have a Ward account and register for access to use this.

Currently this package can perform:
- pickup requests

To create a pickup request:
- Set test or production mode (SetProductionMode()).
- Set shipper information (ShipperInfomation{}).
- Set shipment data (Shipment{}).
- Create the pickup request object (PickupRequest{}).
- Request the pickup (RequestPickup()).
- Check for any errors.
*/
package ward

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

//api urls
const (
	wardTestURL       = "http://208.51.75.23:6082/cgi-bin/map/PICKUPTEST"
	wardProductionURL = "http://208.51.75.23:6082/cgi-bin/map/PICKUP"
)

//wardURL is set to the test URL by default
//This is changed to the production URL when the SetProductionMode function is called
//Forcing the developer to call the SetProductionMode function ensures the production URL is only used
//when actually needed.
var wardURL = wardTestURL

//timeout is the default time we should wait for a reply from Ward
//You may need to adjust this based on how slow connecting to Ward is for you.
//10 seconds is overly long, but sometimes Ward is very slow.
var timeout = time.Duration(10 * time.Second)

//base XML data
var (
	xsiAttr    = "http://www.w3.org/2001/XMLSchema-instance"
	xsdAttr    = "http://www.w3.org/2001/XMLSchema"
	soap12Attr = "http://www.w3.org/2003/05/soap-envelope"
)

//PickupRequest is the main body of the xml request
type PickupRequest struct {
	XMLName xml.Name `xml:"soap12:Envelope"`

	XsiAttr    string `xml:"xmlns:xsi,attr"`    //http://www.w3.org/2001/XMLSchema-instance
	XsdAttr    string `xml:"xmlns:xsd,attr"`    //http://www.w3.org/2001/XMLSchema
	Soap12Attr string `xml:"xmlns:soap12,attr"` //http://www.w3.org/2003/05/soap-envelope

	ShipperInfo ShipperInformation `xml:"soap12:Body>request>ShipperInformation"`
	Shipment    Shipment           `xml:"soap12:Body>request>Shipment"`
}

//ShipperInformation is our ship from address
type ShipperInformation struct {
	ShipperCode                 string //ward account number
	ShipperName                 string //company name
	ShipperAddress1             string
	ShipperAddress2             string
	ShipperCity                 string
	ShipperState                string //xx
	ShipperZipcode              string
	ShipperContactName          string
	ShipperContactTelephone     string //xxxxxxxxxx, only numbers
	ShipperContactEmail         string
	ShipperReadyTime            string //hhmm, 24 hour
	ShipperCloseTime            string //hhmm, 24 hour
	PickupDate                  string //mmddyyyy
	ThirdParty                  string
	ThirdPartyName              string
	ThirdPartyContactName       string
	ThirdPartyContactTelephone  string
	ThirdPartyContactEmail      string
	WardAssuredContactName      string
	WardAssuredContactTelephone string
	WardAssuredContactEmail     string
	ShipperRestriction          string
	DriverNote1                 string
	DriverNote2                 string
	DriverNote3                 string
	RequestOrigin               string //who is making the pickup request
	RequestorUser               string
	RequestorRole               string
	RequestorContactName        string
	RequestorContactTelephone   string
	RequestorContactEmail       string
}

//Shipment is the data on the shipment we are requesting a pickup for
type Shipment struct {
	Pieces                       uint
	PackageCode                  string //code per Ward's website
	Weight                       uint   //lbs
	ConsigneeCode                string
	ConsigneeName                string
	ConsigneeAddress1            string
	ConsigneeAddress2            string
	ConsigneeCity                string
	ConsigneeState               string
	ConsigneeZipcode             string
	ShipperRoutingSCAC           string
	Hazardous                    string //Y or N
	Freezable                    string //Y or N
	DeliveryAppntFlag            string //Y or N
	DeliveryAppntDate            string
	WardAssured12PM              string
	WardAssured03PM              string
	WardAssuredTimeDefinite      string
	WardAssuredTimeDefiniteStart string
	WardAssuredTimeDefiniteEnd   string
	FullValue                    string
	FullValueInsuredAmount       string
	NonStandardSize              string
	NonStandardSizeDescription   string
	RequestorReference           string
	PickupShipmentInstruction1   string
	PickupShipmentInstruction2   string
	PickupShipmentInstruction3   string
	PickupShipmentInstruction4   string
	RequestOrigin                string
}

//Response is the data we get back when a pickup is scheduled successfully
type Response struct {
	XMLName      xml.Name     `xml:"Envelope"`                         //dont need "soap12"
	CreateResult createResult `xml:"Body>CreateResponse>CreateResult"` //dont need "soap12"
}
type createResult struct {
	PickupConfirmation string //the pickup request confirmation number
	Message            string
	PickupTerminal     string
	WardTelephone      string
	WardEmail          string
}

//SetProductionMode chooses the production url for use
func SetProductionMode(yes bool) {
	wardURL = wardProductionURL
	return
}

//SetTimeout updates the timeout value to something the user sets
//use this to increase the timeout if connecting to Ward is really slow
func SetTimeout(seconds time.Duration) {
	timeout = time.Duration(seconds * time.Second)
	return
}

//RequestPickup performs the call to the Ward API to schedule a pickup
func (p *PickupRequest) RequestPickup() (responseData Response, err error) {
	//add xml attributes
	p.XsdAttr = xsdAttr
	p.XsiAttr = xsiAttr
	p.Soap12Attr = soap12Attr

	//convert the pickup request to an xml
	xmlBytes, err := xml.Marshal(p)
	if err != nil {
		err = errors.Wrap(err, "ward.RequestPickup - could not marshal xml")
		return
	}

	//add the xml header and an ending blank line
	//need both to get request to work for some reason
	xmlString := xml.Header + string(xmlBytes) + "\n"

	//make the call to the ward API
	//set a timeout since golang doesn't set one by default and we don't want this to hang forever
	//using application/x-www-form-encoded since this is what Ward's demo used
	httpClient := http.Client{
		Timeout: timeout,
	}
	res, err := httpClient.Post(wardURL, "application/x-www-form-encoded", strings.NewReader(xmlString))
	if err != nil {
		err = errors.Wrap(err, "ward.RequestPickup - could not make post request")
		return
	}

	//read the response
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		err = errors.Wrap(err, "ward.RequestPickup - could not read response 1")
		return
	}

	err = xml.Unmarshal(body, &responseData)
	if err != nil {
		err = errors.Wrap(err, "ward.RequestPickup - could not read response 2")
		return
	}

	//check if data was returned meaning request was successful
	//if not, reread the response data and log it
	if responseData.CreateResult.PickupConfirmation == "" {
		log.Println("ward.RequestPickup - pickup request failed")
		log.Printf(string(body))

		var errorData map[string]interface{}
		xml.Unmarshal(body, &errorData)

		err = errors.New("ward.RequestPickup - pickup request failed")
		log.Println(errorData)
		return
	}

	//pickup request successful
	//response data will have confirmation info
	return
}
