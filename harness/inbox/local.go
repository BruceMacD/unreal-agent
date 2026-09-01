package inbox

import (
	"context"
	"fmt"
)

}


	}
}

	}
	}
	}

	select {
	case <-ctx.Done():
	case <-inputs.ctx.Done():
	}
}

}


	for {
		}

		select {
				continue
			}

		case output <- next:

		case <-inputs.ctx.Done():
			return
		}
	}
}

func cloneInput(input Input) Input {
	input.Payload = input.Payload.Clone()
	return input
}
