package utils

import "errors"

type WeightRandom struct {
	categorys []WeightCategory
	weightSum int
}

func NewWeightRandom(categorys []WeightCategory) (*WeightRandom, error) {
	sum := 0
	for _, v := range categorys {
		sum += v.Weight
	}
	if sum <= 0 {
		return nil, errors.New("Weight sum lte zero")
	}
	return &WeightRandom{categorys: categorys, weightSum: sum}, nil
}

func (this *WeightRandom) Next() *WeightCategory {
	n := Random(0, this.weightSum)
	m := 0
	for _, v := range this.categorys {
		if m <= n && n < m+v.Weight {
			return &v
		}
		m += v.Weight
	}
	return nil
}

type WeightCategory struct {
	Category interface{}
	Weight   int
}

type CategoryProportion struct {
	Category   interface{}
	Proportion float64
}

type CategoryProportions []CategoryProportion

func (p CategoryProportions) Len() int {
	return len(p)
}

func (p CategoryProportions) Less(i, j int) bool {
	return p[i].Proportion < p[j].Proportion
}

func (p CategoryProportions) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
