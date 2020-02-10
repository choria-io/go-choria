package confkey

type Doc struct {
	description string
	url         string
	deprecated  bool
	structKey   string
	configKey   string
	container   string
	dflt        string
	env         string
	vtype       string
	validation  string
}

// Deprecated indicates if the item is not in use anymore
func (d *Doc) Deprecate() bool {
	return d.deprecated
}

// StructKey is the key within the structure to lookup to retrieve the item
func (d *Doc) StructKey() string {
	if d.container != "" {
		return d.container + "." + d.structKey
	}

	return d.structKey
}

// ConfigKey is the key to place within the configuration to set the item
func (d *Doc) ConfigKey() string {
	return d.configKey
}

// Type is the type of data to store in the item
func (d *Doc) Type() string {
	return d.vtype
}

// Description is a description of the item, empty when not set
func (d *Doc) Description() string {
	if d.description == "" {
		return "Undocumented"
	}

	return d.description
}

// SetDescription overrides the description of the key
func (d *Doc) SetDescription(desc string) {
	d.description = desc
}

// URL returns a url that describes the related feature in more detail
func (d *Doc) URL() string {
	return d.url
}

// Default is the default value as a string
func (d *Doc) Default() string {
	return d.dflt
}

// Validation is the configured validation
func (d *Doc) Validation() string {
	return d.validation
}

// Environment is an environment variable that can set this item, empty when not settable
func (d *Doc) Environment() string {
	return d.env
}

// KeyDoc constructs a Doc for key within target, marked up to be within container
func KeyDoc(target interface{}, key string, container string) *Doc {
	var err error

	d := Doc{
		configKey: key,
		container: container,
	}

	d.structKey, err = FieldWithKey(target, key)
	if err != nil {
		return nil
	}

	d.description, _ = Description(target, key)
	d.url, _ = URL(target, key)
	d.dflt, _ = DefaultString(target, key)
	d.env, _ = Environment(target, key)
	d.vtype, _ = Type(target, key)
	d.validation, _ = Validation(target, key)
	d.deprecated, _ = IsDeprecated(target, key)

	return &d
}
