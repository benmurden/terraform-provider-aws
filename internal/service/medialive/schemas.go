package medialive

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func destinationSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"destination_ref_id": {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

func connectionRetryIntervalSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeInt,
		Optional: true,
		Computed: true,
	}
}

func filecacheDurationSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeInt,
		Optional: true,
		Computed: true,
	}
}

func numRetriesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeInt,
		Optional: true,
		Computed: true,
	}
}

func restartDelaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeInt,
		Optional: true,
		Computed: true,
	}
}