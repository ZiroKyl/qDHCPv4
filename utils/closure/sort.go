package closure

import "sort"

type SortS struct{
	a []interface{};
	closure func(i0, i1 int) bool;
}

func (s SortS) Len() int             { return len(s.a); }
func (s SortS) Swap(i0, i1 int)      { s.a[i0],s.a[i1] = s.a[i1],s.a[i0]; }
func (s SortS) Less(i0, i1 int) bool { return s.closure(i0, i1); }

func Sort(a []interface{}, closure func(i0, i1 int) bool){
	sort.Sort(SortS{a, closure});
}