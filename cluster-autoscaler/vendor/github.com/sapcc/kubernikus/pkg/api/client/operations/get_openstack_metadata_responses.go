// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/sapcc/kubernikus/pkg/api/models"
)

// GetOpenstackMetadataReader is a Reader for the GetOpenstackMetadata structure.
type GetOpenstackMetadataReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetOpenstackMetadataReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewGetOpenstackMetadataOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewGetOpenstackMetadataDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetOpenstackMetadataOK creates a GetOpenstackMetadataOK with default headers values
func NewGetOpenstackMetadataOK() *GetOpenstackMetadataOK {
	return &GetOpenstackMetadataOK{}
}

/*GetOpenstackMetadataOK handles this case with default header values.

OK
*/
type GetOpenstackMetadataOK struct {
	Payload *models.OpenstackMetadata
}

func (o *GetOpenstackMetadataOK) Error() string {
	return fmt.Sprintf("[GET /api/v1/openstack/metadata][%d] getOpenstackMetadataOK  %+v", 200, o.Payload)
}

func (o *GetOpenstackMetadataOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.OpenstackMetadata)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetOpenstackMetadataDefault creates a GetOpenstackMetadataDefault with default headers values
func NewGetOpenstackMetadataDefault(code int) *GetOpenstackMetadataDefault {
	return &GetOpenstackMetadataDefault{
		_statusCode: code,
	}
}

/*GetOpenstackMetadataDefault handles this case with default header values.

Error
*/
type GetOpenstackMetadataDefault struct {
	_statusCode int

	Payload *models.Error
}

// Code gets the status code for the get openstack metadata default response
func (o *GetOpenstackMetadataDefault) Code() int {
	return o._statusCode
}

func (o *GetOpenstackMetadataDefault) Error() string {
	return fmt.Sprintf("[GET /api/v1/openstack/metadata][%d] GetOpenstackMetadata default  %+v", o._statusCode, o.Payload)
}

func (o *GetOpenstackMetadataDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
