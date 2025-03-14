package ns1

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/fatih/structs"
	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
	"gopkg.in/ns1/ns1-go.v2/rest/model/filter"
)

var recordTypeStringEnum = NewStringEnum([]string{
	"A",
	"AAAA",
	"ALIAS",
	"AFSDB",
	"CNAME",
	"DNAME",
	"HINFO",
	"MX",
	"NAPTR",
	"NS",
	"PTR",
	"RP",
	"SPF",
	"SRV",
	"TXT",
})

func recordResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			// Required
			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"domain": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: recordTypeStringEnum.ValidateFunc,
			},
			// Optional
			"ttl": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"meta": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"link": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"use_client_subnet": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"short_answers": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"answers": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"answer": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"region": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"meta": {
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
			"regions": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"meta": {
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
			"filters": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"filter": {
							Type:     schema.TypeString,
							Required: true,
						},
						"disabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"config": {
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
		},
		Create:   RecordCreate,
		Read:     RecordRead,
		Update:   RecordUpdate,
		Delete:   RecordDelete,
		Importer: &schema.ResourceImporter{State: recordStateFunc},
	}
}

// errJoin joins errors into a single error
func errJoin(errs []error, sep string) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	case 2:
		// Special case for common small values.
		// Remove if golang.org/issue/6714 is fixed
		return errors.New(errs[0].Error() + sep + errs[1].Error())
	case 3:
		// Same special case
		return errors.New(errs[0].Error() + sep + errs[1].Error() + sep + errs[2].Error())
	}

	n := len(sep) * (len(errs) - 1)
	for i := 0; i < len(errs); i++ {
		n += len(errs[i].Error())
	}

	b := make([]byte, n)
	bp := copy(b, errs[0].Error())
	for _, err := range errs[1:] {
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], err.Error())
	}
	return errors.New(string(b))
}

func recordToResourceData(d *schema.ResourceData, r *dns.Record) error {
	d.SetId(r.ID)
	d.Set("domain", r.Domain)
	d.Set("zone", r.Zone)
	d.Set("type", r.Type)
	d.Set("ttl", r.TTL)
	if r.Link != "" {
		d.Set("link", r.Link)
	}

	// top level meta works but nested meta doesn't
	if r.Meta != nil {
		d.Set("meta", structs.Map(r.Meta))
	}
	if r.UseClientSubnet != nil {
		d.Set("use_client_subnet", *r.UseClientSubnet)
	}
	if len(r.Filters) > 0 {
		filters := make([]map[string]interface{}, len(r.Filters))
		for i, f := range r.Filters {
			m := make(map[string]interface{})
			m["filter"] = f.Type
			if f.Disabled {
				m["disabled"] = true
			}
			if f.Config != nil {
				m["config"] = f.Config
			}
			filters[i] = m
		}
		d.Set("filters", filters)
	}
	if len(r.Answers) > 0 {
		ans := make([]map[string]interface{}, 0)
		log.Printf("Got back from ns1 answers: %+v", r.Answers)
		for _, answer := range r.Answers {
			ans = append(ans, answerToMap(*answer))
		}
		log.Printf("Setting answers %+v", ans)
		err := d.Set("answers", ans)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting answers for: %s, error: %#v", r.Domain, err)
		}
	}
	if len(r.Regions) > 0 {
		keys := make([]string, 0, len(r.Regions))
		for regionName := range r.Regions {
			keys = append(keys, regionName)
		}
		sort.Strings(keys)
		regions := make([]map[string]interface{}, 0, len(r.Regions))
		for _, k := range keys {
			newRegion := make(map[string]interface{})
			region := r.Regions[k]
			newRegion["name"] = k
			newRegion["meta"] = region.Meta.StringMap()
			regions = append(regions, newRegion)
		}
		log.Printf("Setting regions %+v", regions)
		err := d.Set("regions", regions)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting regions for: %s, error: %#v", r.Domain, err)
		}
	}
	return nil
}

func answerToMap(a dns.Answer) map[string]interface{} {
	m := make(map[string]interface{})
	m["answer"] = strings.Join(a.Rdata, " ")
	if a.RegionName != "" {
		m["region"] = a.RegionName
	}
	if a.Meta != nil {
		log.Println("got meta: ", a.Meta)
		m["meta"] = a.Meta.StringMap()
		log.Println(m["meta"])
	}
	return m
}

func resourceDataToRecord(r *dns.Record, d *schema.ResourceData) error {
	r.ID = d.Id()
	log.Printf("answers from template: %+v, %T\n", d.Get("answers"), d.Get("answers"))

	if shortAnswers := d.Get("short_answers").([]interface{}); len(shortAnswers) > 0 {
		for _, answerRaw := range shortAnswers {
			answer := answerRaw.(string)
			switch d.Get("type") {
			case "TXT", "SPF":
				r.AddAnswer(dns.NewTXTAnswer(answer))
			default:
				r.AddAnswer(dns.NewAnswer(strings.Split(answer, " ")))
			}
		}
	}
	if answers := d.Get("answers").([]interface{}); len(answers) > 0 {
		for _, answerRaw := range answers {
			answer := answerRaw.(map[string]interface{})
			var a *dns.Answer

			v := answer["answer"].(string)
			switch d.Get("type") {
			case "TXT", "SPF":
				a = dns.NewTXTAnswer(v)
			default:
				a = dns.NewAnswer(strings.Split(v, " "))
			}

			if v, ok := answer["region"]; ok {
				a.RegionName = v.(string)
			}

			if v, ok := answer["meta"]; ok {
				log.Println("answer meta", v)
				a.Meta = data.MetaFromMap(v.(map[string]interface{}))
				log.Println(a.Meta)
				errs := a.Meta.Validate()
				if len(errs) > 0 {
					return errJoin(append([]error{errors.New("found error/s in answer metadata")}, errs...), ",")
				}
			}

			r.AddAnswer(a)
		}
	}
	log.Println("number of answers found:", len(r.Answers))

	if v, ok := d.GetOk("ttl"); ok {
		r.TTL = v.(int)
	}
	if v, ok := d.GetOk("link"); ok {
		if len(r.Answers) > 0 {
			return errors.New("cannot have both link and answers in a record")
		}
		r.LinkTo(v.(string))
	}

	if v, ok := d.GetOk("meta"); ok {
		log.Println("record meta", v)
		r.Meta = data.MetaFromMap(v.(map[string]interface{}))
		log.Println(r.Meta)
		errs := r.Meta.Validate()
		if len(errs) > 0 {
			return errJoin(append([]error{errors.New("found error/s in record metadata")}, errs...), ",")
		}
	}
	useClientSubnet := d.Get("use_client_subnet").(bool)
	r.UseClientSubnet = &useClientSubnet

	if rawFilters := d.Get("filters").([]interface{}); len(rawFilters) > 0 {
		filters := make([]*filter.Filter, len(rawFilters))
		for i, filterRaw := range rawFilters {
			fi := filterRaw.(map[string]interface{})
			config := make(map[string]interface{})
			f := filter.Filter{
				Type:   fi["filter"].(string),
				Config: config,
			}
			if disabled, ok := fi["disabled"]; ok {
				f.Disabled = disabled.(bool)
			}
			if rawConfig, ok := fi["config"]; ok {
				f.Config = rawConfig.(map[string]interface{})
			}
			filters[i] = &f
		}
		r.Filters = filters
	}
	if regions := d.Get("regions").([]interface{}); len(regions) > 0 {
		for _, regionRaw := range regions {
			region := regionRaw.(map[string]interface{})
			ns1R := data.Region{
				Meta: data.Meta{},
			}

			if v, ok := region["meta"]; ok {
				log.Println("region meta", v)
				meta := data.MetaFromMap(v.(map[string]interface{}))
				log.Println("region meta object", meta)
				ns1R.Meta = *meta
				log.Println(ns1R.Meta)
				errs := ns1R.Meta.Validate()
				if len(errs) > 0 {
					return errJoin(append([]error{errors.New("found error/s in region/group metadata")}, errs...), ",")
				}
			}
			r.Regions[region["name"].(string)] = ns1R
		}
	}
	return nil
}

// RecordCreate creates DNS record in ns1
func RecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	r := dns.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err := resourceDataToRecord(r, d); err != nil {
		return err
	}
	if _, err := client.Records.Create(r); err != nil {
		return err
	}
	return recordToResourceData(d, r)
}

// RecordRead reads the DNS record from ns1
func RecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	r, _, err := client.Records.Get(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err != nil {
		return err
	}

	return recordToResourceData(d, r)
}

// RecordDelete deletes the DNS record from ns1
func RecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	_, err := client.Records.Delete(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	d.SetId("")
	return err
}

// RecordUpdate updates the given dns record in ns1
func RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	r := dns.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err := resourceDataToRecord(r, d); err != nil {
		return err
	}
	if _, err := client.Records.Update(r); err != nil {
		return err
	}
	return recordToResourceData(d, r)
}

func recordStateFunc(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid record specifier.  Expecting 2 slashes (\"zone/domain/type\"), got %d", len(parts)-1)
	}

	d.Set("zone", parts[0])
	d.Set("domain", parts[1])
	d.Set("type", parts[2])

	return []*schema.ResourceData{d}, nil
}
