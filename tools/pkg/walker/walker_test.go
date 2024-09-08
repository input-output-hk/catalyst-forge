package walker

import "github.com/spf13/afero"

type wrapfs struct {
	afero.Fs

	attempts  int
	failAfter int
	trigger   error
}

func (w *wrapfs) Open(name string) (afero.File, error) {
	w.attempts++
	if w.attempts == w.failAfter {
		return nil, w.trigger
	}
	return afero.Fs.Open(w.Fs, name)
}
