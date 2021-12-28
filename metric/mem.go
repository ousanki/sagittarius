package metric

import (
	"bytes"
	"fmt"
	"github.com/ousanki/sagittarius/core/log"
	"runtime"
	"time"
)

type MemCycleData struct {
	// 周期内堆分配总量 只计算增加
	AllocTotal string
	// 周期内golang分配堆变化
	HeapAlloc string
	// 周期内系统分配堆变化
	HeapSys string
	// 周期栈用量变化
	StackInuse string
	// 周期内系统分配栈变化
	StackSys string
	// 周期内GC导致暂停时常(纳秒)
	PauseTotalNs uint64
	// 周期内malloc次数
	Mallocs uint64
	// 周期内完成GC次数
	NumGC uint32
	// 周期内goroutine数量变化
	NumGoroutine string
}

type MemData struct {
	// 堆分配总量 只计算增加
	AllocTotal string
	// golang分配堆数量
	HeapAlloc string
	// 系统分配堆数量
	HeapSys string
	// malloc次数
	Mallocs uint64
	// 栈用量
	StackInuse string
	// 系统分配栈数量
	StackSys string
	// GC导致暂停时常(纳秒)
	PauseTotalNs uint64
	// 完成GC次数
	NumGC uint32
	// GC消耗CPU情况
	GCCPUFraction string
	// goroutine数量
	NumGoroutine string
}

type ReportData struct {
	MemCycleData
	MemData
}

func (r *ReportData) Format() string {
	var buf bytes.Buffer
	buf.WriteString("|||||||||||||||||||||||||||||||||||||||||||||||||\n")
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString("|               |    Current    |     Cycle     |\n")
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|   AllocTotal  |%15s|%15s|\n", r.MemData.AllocTotal, r.MemCycleData.AllocTotal))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|   HeapAlloc   |%15s|%15s|\n", r.MemData.HeapAlloc, r.MemCycleData.HeapAlloc))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|    HeapSys    |%15s|%15s|\n", r.MemData.HeapSys, r.MemCycleData.HeapSys))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|  StackInuse   |%15s|%15s|\n", r.MemData.StackInuse, r.MemCycleData.StackInuse))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|   StackSys    |%15s|%15s|\n", r.MemData.StackSys, r.MemCycleData.StackSys))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("| PauseTotalNs  |%15d|%15d|\n", r.MemData.PauseTotalNs, r.MemCycleData.PauseTotalNs))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|     NumGC     |%15d|%15d|\n", r.MemData.NumGC, r.MemCycleData.NumGC))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("| NumGoroutine  |%15s|%15s|\n", r.MemData.NumGoroutine, r.MemCycleData.NumGoroutine))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("| GCCPUFraction |%15s|               |\n", r.MemData.GCCPUFraction))
	buf.WriteString("|---------------|---------------|---------------|\n\n")
	return buf.String()
}

type MemMetricData struct {
	runtime.MemStats
	NumGoroutine int
}

type MemMetric struct {
	before  MemMetricData
	current MemMetricData
}

func (mm *MemMetric) GetReport() ReportData {
	return ReportData{
		MemCycleData: mm.getMemCycle(),
		MemData:      mm.getMem(),
	}
}

func (mm *MemMetric) getMemCycle() MemCycleData {
	d := MemCycleData{
		AllocTotal:   size2Human(mm.current.TotalAlloc - mm.before.TotalAlloc),
		PauseTotalNs: mm.current.PauseTotalNs - mm.before.PauseTotalNs,
		Mallocs:      mm.current.Mallocs - mm.before.Mallocs,
		NumGC:        mm.current.NumGC - mm.before.NumGC,
	}
	if mm.current.HeapAlloc >= mm.before.HeapAlloc {
		d.HeapAlloc = size2Human(mm.current.HeapAlloc - mm.before.HeapAlloc)
	} else {
		d.HeapAlloc = fmt.Sprintf("-%s", size2Human(mm.before.HeapAlloc-mm.current.HeapAlloc))
	}
	if mm.current.HeapSys >= mm.before.HeapSys {
		d.HeapSys = size2Human(mm.current.HeapSys - mm.before.HeapSys)
	} else {
		d.HeapSys = fmt.Sprintf("-%s", size2Human(mm.before.HeapSys-mm.current.HeapSys))
	}
	if mm.current.StackInuse >= mm.before.StackInuse {
		d.StackInuse = size2Human(mm.current.StackInuse - mm.before.StackInuse)
	} else {
		d.StackInuse = fmt.Sprintf("-%s", size2Human(mm.before.StackInuse-mm.current.StackInuse))
	}
	if mm.current.StackSys >= mm.before.StackSys {
		d.StackSys = size2Human(mm.current.StackSys - mm.before.StackSys)
	} else {
		d.StackSys = fmt.Sprintf("-%s", size2Human(mm.before.StackSys-mm.current.StackSys))
	}
	if mm.current.NumGoroutine >= mm.before.NumGoroutine {
		d.NumGoroutine = fmt.Sprintf("%d", mm.current.NumGoroutine - mm.before.NumGoroutine)
	} else {
		d.NumGoroutine = fmt.Sprintf("-%d", mm.before.NumGoroutine-mm.current.NumGoroutine)
	}
	return d
}

func (mm *MemMetric) getMem() MemData {
	return MemData{
		AllocTotal:    size2Human(mm.current.TotalAlloc),
		HeapAlloc:     size2Human(mm.current.HeapAlloc),
		HeapSys:       size2Human(mm.current.HeapSys),
		StackInuse:    size2Human(mm.current.StackInuse),
		StackSys:      size2Human(mm.current.StackSys),
		PauseTotalNs:  mm.current.PauseTotalNs,
		Mallocs:       mm.current.Mallocs,
		NumGC:         mm.current.NumGC,
		GCCPUFraction: fmt.Sprintf("%.2f%%", mm.current.GCCPUFraction*100),
		NumGoroutine:  fmt.Sprintf("%d", mm.current.NumGoroutine),
	}
}

func size2Human(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.2fK", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2fM", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2fG", float64(size)/(1024*1024*1024))
}

var _m *MemMetric
var stackLogger *log.Logger

func init() {
	_m = new(MemMetric)
	stackLogger = log.New("stack")
	stackLogger.WithOptions(
		log.SetRotation(log.RotationDay),
		log.SetPath("./log"),
		log.SetFormat(log.ConsoleFormat),
	)
}

func loadStats() {
	runtime.ReadMemStats(&_m.current.MemStats)
	_m.current.NumGoroutine = runtime.NumGoroutine()
}

func Metric() {
	go func() {
		for {
			<-time.After(time.Second)
			// 读取信息
			loadStats()
			// 生成报告
			report := _m.GetReport()
			stackLogger.Writeln(report.Format())
			// 后处理
			_m.before = _m.current
		}
	}()
}
