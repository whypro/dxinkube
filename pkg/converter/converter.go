package converter

type AddrConverterInterface interface {
	ConvertAddr(podAddr string) (string, error)
	Run(stopCh <-chan struct{})
}
