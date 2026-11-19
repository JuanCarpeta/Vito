package main

// Configuraciones especificas de Windows para lanzar procesos sin consola.

import (
	"syscall"
)

func sysProcOculto() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
