package {package}

import (
	"github.com/pkg/errors"
)

var _ = errors.Wrap

{foreach $calls as $call}
func (*{name}) {call.name|capitalize}(r *{name|lcfirst}{call.name|capitalize}Request) (interface{}, error) {
	return nil, errors.New("Not implemented: {name}.{call.name}")
}

{/foreach}
