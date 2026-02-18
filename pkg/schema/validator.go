package schema

import "fmt"

// Validate checks a CompiledSchema for consistency.
// Returns a slice of ValidationErrors (empty = valid).
func Validate(cs *CompiledSchema) []error {
	var errs []error

	for typeName, td := range cs.Types {
		// Validate relation type references exist
		for relName, rd := range td.Relations {
			for _, ref := range rd.AllowedTypes {
				if _, ok := cs.Types[ref.Type]; !ok {
					errs = append(errs, fmt.Errorf("type %q relation %q: references unknown type %q",
						typeName, relName, ref.Type))
				}
				if ref.Relation != "" {
					refType := cs.Types[ref.Type]
					if refType != nil {
						if _, ok := refType.Relations[ref.Relation]; !ok {
							errs = append(errs, fmt.Errorf("type %q relation %q: type %q has no relation %q",
								typeName, relName, ref.Type, ref.Relation))
						}
					}
				}
			}
		}

		// Validate permission references resolve
		for permName, pd := range td.Permissions {
			for _, child := range pd.Children {
				if err := validatePermRef(cs, typeName, permName, child); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	return errs
}

func validatePermRef(cs *CompiledSchema, typeName, permName string, ref *PermissionRef) error {
	td := cs.Types[typeName]
	switch ref.Kind() {
	case "relation":
		if _, ok := td.Relations[ref.RelationRef]; !ok {
			// Could also be a permission reference (computed usersets)
			if _, ok := td.Permissions[ref.RelationRef]; !ok {
				return fmt.Errorf("type %q permission %q: references unknown relation/permission %q",
					typeName, permName, ref.RelationRef)
			}
		}
	case "arrow":
		if _, ok := td.Relations[ref.ArrowRef]; !ok {
			return fmt.Errorf("type %q permission %q: arrow references unknown relation %q",
				typeName, permName, ref.ArrowRef)
		}
	}
	return nil
}
