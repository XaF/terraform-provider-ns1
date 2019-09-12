package ns1

import (
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func dataSourceZone() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"zone": {
				Type:     schema.TypeString,
				Required: true,
			},
			"ttl": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"refresh": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"retry": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"expiry": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"nx_ttl": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"link": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"additional_primaries": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"dns_servers": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hostmaster": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"networks": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeInt},
			},
			"secondaries": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Type:     schema.TypeString,
							Required: true,
						},
						"notify": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"network_ids": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeInt},
						},
					},
				},
			},
		},
		Read: dataSourceZoneRead,
	}
}

func dataSourceZoneToResourceData(d *schema.ResourceData, z *dns.Zone) {
	d.SetId(z.ID)
	d.Set("hostmaster", z.Hostmaster)
	d.Set("ttl", z.TTL)
	d.Set("nx_ttl", z.NxTTL)
	d.Set("refresh", z.Refresh)
	d.Set("retry", z.Retry)
	d.Set("expiry", z.Expiry)
	d.Set("networks", z.NetworkIDs)
	d.Set("dns_servers", strings.Join(z.DNSServers[:], ","))
	d.Set("link", z.Link)
	if z.Secondary != nil && z.Secondary.Enabled {
		d.Set("primary", z.Secondary.PrimaryIP)
		d.Set("additional_primaries", z.Secondary.OtherIPs)
	}
	if z.Primary != nil && z.Primary.Enabled {
		d.Set("secondaries", z.Primary.Secondaries)
	}
}

func dataSourceZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	z, _, err := client.Zones.Get(d.Get("zone").(string))
	if err != nil {
		return err
	}
	dataSourceZoneToResourceData(d, z)
	return nil
}
