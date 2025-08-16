package ai

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// BehaviorAnalyzer represents an AI-powered behavior analysis system
type BehaviorAnalyzer struct {
	models      map[string]*MLModel
	modelsMutex sync.RWMutex
	features    *FeatureExtractor
	classifier  *AnomalyClassifier
	config      *BehaviorConfig
	metrics     *BehaviorMetrics
}

// MLModel represents a machine learning model
type MLModel struct {
	Name        string
	Model       interface{} // TensorFlow, PyTorch, etc.
	Version     string
	Accuracy    float64
	LastUpdated time.Time
	IsTrained   bool
}

// FeatureExtractor represents a feature extraction system
type FeatureExtractor struct {
	extractors map[string]FeatureExtractorFunc
	config     *FeatureConfig
}

// FeatureExtractorFunc represents a feature extraction function
type FeatureExtractorFunc func(data interface{}) ([]float64, error)

// FeatureConfig represents configuration for feature extraction
type FeatureConfig struct {
	WindowSize     int
	FeatureCount   int
	NormalizeData  bool
	EnableCaching  bool
	CacheTTL       time.Duration
}

// AnomalyClassifier represents an anomaly classification system
type AnomalyClassifier struct {
	models    map[string]*AnomalyModel
	threshold float64
	config    *ClassifierConfig
}

// AnomalyModel represents an anomaly detection model
type AnomalyModel struct {
	Name      string
	Algorithm string // Isolation Forest, LOF, etc.
	Model     interface{}
	Threshold float64
	Trained   bool
}

// ClassifierConfig represents configuration for anomaly classification
type ClassifierConfig struct {
	DefaultThreshold float64
	EnableEnsemble   bool
	ModelCount       int
	UpdateInterval   time.Duration
}

// BehaviorConfig represents configuration for behavior analysis
type BehaviorConfig struct {
	AnalysisInterval time.Duration
	ModelPath        string
	InferenceTimeout time.Duration
	EnableRealTime   bool
	BatchSize        int
}

// BehaviorData represents data for behavior analysis
type BehaviorData struct {
	UserID      string
	Timestamp   time.Time
	Actions     []string
	Metrics     map[string]float64
	Context     map[string]interface{}
	Source      string
}

// BehaviorAnalysis represents the result of behavior analysis
type BehaviorAnalysis struct {
	Classification string
	Anomalies      []Anomaly
	Confidence     float64
	Timestamp      time.Time
	Features       []float64
	RiskScore      float64
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Index     int
	Score     float64
	Model     string
	Severity  string
	Timestamp time.Time
	Details   map[string]interface{}
}

// BehaviorMetrics represents metrics for behavior analysis
type BehaviorMetrics struct {
	TotalAnalyses     int64
	AnomaliesDetected int64
	AverageConfidence float64
	ProcessingTime    time.Duration
	ModelAccuracy     float64
	LastAnalysis      time.Time
}

// NewBehaviorAnalyzer creates a new behavior analyzer
func NewBehaviorAnalyzer(config *BehaviorConfig) *BehaviorAnalyzer {
	if config == nil {
		config = &BehaviorConfig{
			AnalysisInterval: 5 * time.Second,
			ModelPath:        "/etc/cloudbridge/models",
			InferenceTimeout: 10 * time.Second,
			EnableRealTime:   true,
			BatchSize:        100,
		}
	}

	return &BehaviorAnalyzer{
		models:     make(map[string]*MLModel),
		features:   NewFeatureExtractor(nil),
		classifier: NewAnomalyClassifier(nil),
		config:     config,
		metrics:    &BehaviorMetrics{},
	}
}

// NewFeatureExtractor creates a new feature extractor
func NewFeatureExtractor(config *FeatureConfig) *FeatureExtractor {
	if config == nil {
		config = &FeatureConfig{
			WindowSize:    100,
			FeatureCount:  50,
			NormalizeData: true,
			EnableCaching: true,
			CacheTTL:      1 * time.Hour,
		}
	}

	return &FeatureExtractor{
		extractors: make(map[string]FeatureExtractorFunc),
		config:     config,
	}
}

// NewAnomalyClassifier creates a new anomaly classifier
func NewAnomalyClassifier(config *ClassifierConfig) *AnomalyClassifier {
	if config == nil {
		config = &ClassifierConfig{
			DefaultThreshold: 0.8,
			EnableEnsemble:   true,
			ModelCount:       3,
			UpdateInterval:   1 * time.Hour,
		}
	}

	return &AnomalyClassifier{
		models:    make(map[string]*AnomalyModel),
		threshold: config.DefaultThreshold,
		config:    config,
	}
}

// AnalyzeBehavior analyzes behavior data and returns analysis results
func (ba *BehaviorAnalyzer) AnalyzeBehavior(data *BehaviorData) (*BehaviorAnalysis, error) {
	startTime := time.Now()

	// Extract features
	features, err := ba.features.Extract(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract features: %w", err)
	}

	// Classify behavior
	classification, err := ba.classifier.Classify(features)
	if err != nil {
		return nil, fmt.Errorf("failed to classify behavior: %w", err)
	}

	// Detect anomalies
	anomalies, err := ba.detectAnomalies(features)
	if err != nil {
		return nil, fmt.Errorf("failed to detect anomalies: %w", err)
	}

	// Calculate confidence and risk score
	confidence := ba.calculateConfidence(features)
	riskScore := ba.calculateRiskScore(anomalies, confidence)

	analysis := &BehaviorAnalysis{
		Classification: classification,
		Anomalies:     anomalies,
		Confidence:    confidence,
		Timestamp:     time.Now(),
		Features:      features,
		RiskScore:     riskScore,
	}

	// Update metrics
	ba.updateMetrics(analysis, time.Since(startTime))

	return analysis, nil
}

// Extract extracts features from behavior data
func (fe *FeatureExtractor) Extract(data *BehaviorData) ([]float64, error) {
	var features []float64

	// Extract basic features
	basicFeatures, err := fe.extractBasicFeatures(data)
	if err != nil {
		return nil, err
	}
	features = append(features, basicFeatures...)

	// Extract temporal features
	temporalFeatures, err := fe.extractTemporalFeatures(data)
	if err != nil {
		return nil, err
	}
	features = append(features, temporalFeatures...)

	// Extract contextual features
	contextualFeatures, err := fe.extractContextualFeatures(data)
	if err != nil {
		return nil, err
	}
	features = append(features, contextualFeatures...)

	// Normalize features if enabled
	if fe.config.NormalizeData {
		features = fe.normalizeFeatures(features)
	}

	return features, nil
}

// extractBasicFeatures extracts basic features from behavior data
func (fe *FeatureExtractor) extractBasicFeatures(data *BehaviorData) ([]float64, error) {
	var features []float64

	// Action count
	features = append(features, float64(len(data.Actions)))

	// Unique actions
	uniqueActions := make(map[string]bool)
	for _, action := range data.Actions {
		uniqueActions[action] = true
	}
	features = append(features, float64(len(uniqueActions)))

	// Action frequency
	if len(data.Actions) > 0 {
		features = append(features, float64(len(data.Actions))/float64(fe.config.WindowSize))
	} else {
		features = append(features, 0.0)
	}

	// Metrics features
	for _, value := range data.Metrics {
		features = append(features, value)
	}

	return features, nil
}

// extractTemporalFeatures extracts temporal features from behavior data
func (fe *FeatureExtractor) extractTemporalFeatures(data *BehaviorData) ([]float64, error) {
	var features []float64

	// Time-based features
	hour := float64(data.Timestamp.Hour())
	features = append(features, hour/24.0) // Normalized hour

	weekday := float64(data.Timestamp.Weekday())
	features = append(features, weekday/7.0) // Normalized weekday

	// Time since last action (if available)
	// This would require historical data, so we'll use a placeholder
	features = append(features, 0.0)

	return features, nil
}

// extractContextualFeatures extracts contextual features from behavior data
func (fe *FeatureExtractor) extractContextualFeatures(data *BehaviorData) ([]float64, error) {
	var features []float64

	// Source-based features
	if data.Source != "" {
		// Simple hash-based feature for source
		hash := 0
		for _, char := range data.Source {
			hash = (hash*31 + int(char)) % 1000
		}
		features = append(features, float64(hash)/1000.0)
	} else {
		features = append(features, 0.0)
	}

	// Context features
	for _, value := range data.Context {
		switch v := value.(type) {
		case float64:
			features = append(features, v)
		case int:
			features = append(features, float64(v))
		case bool:
			if v {
				features = append(features, 1.0)
			} else {
				features = append(features, 0.0)
			}
		default:
			features = append(features, 0.0)
		}
	}

	return features, nil
}

// normalizeFeatures normalizes features to 0-1 range
func (fe *FeatureExtractor) normalizeFeatures(features []float64) []float64 {
	if len(features) == 0 {
		return features
	}

	// Find min and max values
	min := features[0]
	max := features[0]
	for _, f := range features {
		if f < min {
			min = f
		}
		if f > max {
			max = f
		}
	}

	// Normalize
	normalized := make([]float64, len(features))
	range_ := max - min
	if range_ == 0 {
		// All values are the same, set to 0.5
		for i := range normalized {
			normalized[i] = 0.5
		}
	} else {
		for i, f := range features {
			normalized[i] = (f - min) / range_
		}
	}

	return normalized
}

// Classify classifies behavior based on features
func (ac *AnomalyClassifier) Classify(features []float64) (string, error) {
	// Simple classification based on feature values
	// In a real implementation, you would use trained ML models
	
	// Calculate average feature value
	sum := 0.0
	for _, f := range features {
		sum += f
	}
	average := sum / float64(len(features))

	// Simple classification logic
	if average < 0.3 {
		return "normal", nil
	} else if average < 0.7 {
		return "suspicious", nil
	} else {
		return "anomalous", nil
	}
}

// detectAnomalies detects anomalies in the features
func (ba *BehaviorAnalyzer) detectAnomalies(features []float64) ([]Anomaly, error) {
	var anomalies []Anomaly

	// Simple anomaly detection based on statistical methods
	// In a real implementation, you would use sophisticated ML models
	
	// Calculate mean and standard deviation
	mean := ba.calculateMean(features)
	stdDev := ba.calculateStdDev(features, mean)

	// Detect outliers (values more than 2 standard deviations from mean)
	for i, feature := range features {
		zScore := math.Abs((feature - mean) / stdDev)
		if zScore > 2.0 {
			anomaly := Anomaly{
				Index:     i,
				Score:     zScore,
				Model:     "statistical",
				Severity:  ba.getSeverity(zScore),
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"z_score":    zScore,
					"mean":       mean,
					"std_dev":    stdDev,
					"feature_id": i,
				},
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies, nil
}

// calculateMean calculates the mean of a slice of floats
func (ba *BehaviorAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStdDev calculates the standard deviation of a slice of floats
func (ba *BehaviorAnalyzer) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return math.Sqrt(sum / float64(len(values)))
}

// getSeverity returns the severity level based on z-score
func (ba *BehaviorAnalyzer) getSeverity(zScore float64) string {
	if zScore > 3.0 {
		return "critical"
	} else if zScore > 2.5 {
		return "high"
	} else if zScore > 2.0 {
		return "medium"
	} else {
		return "low"
	}
}

// calculateConfidence calculates the confidence level of the analysis
func (ba *BehaviorAnalyzer) calculateConfidence(features []float64) float64 {
	// Simple confidence calculation based on feature quality
	// In a real implementation, you would use model confidence scores
	
	if len(features) == 0 {
		return 0.0
	}

	// Calculate feature variance as a proxy for confidence
	mean := ba.calculateMean(features)
	variance := 0.0
	for _, f := range features {
		variance += math.Pow(f-mean, 2)
	}
	variance /= float64(len(features))

	// Higher variance = lower confidence
	confidence := 1.0 - math.Min(variance, 1.0)
	return math.Max(confidence, 0.0)
}

// calculateRiskScore calculates the overall risk score
func (ba *BehaviorAnalyzer) calculateRiskScore(anomalies []Anomaly, confidence float64) float64 {
	// Calculate risk based on anomalies and confidence
	riskScore := 0.0

	// Add risk from anomalies
	for _, anomaly := range anomalies {
		switch anomaly.Severity {
		case "critical":
			riskScore += 0.4
		case "high":
			riskScore += 0.2
		case "medium":
			riskScore += 0.1
		case "low":
			riskScore += 0.05
		}
	}

	// Adjust based on confidence
	riskScore *= (2.0 - confidence) // Lower confidence = higher risk

	return math.Min(riskScore, 1.0)
}

// updateMetrics updates the behavior analysis metrics
func (ba *BehaviorAnalyzer) updateMetrics(analysis *BehaviorAnalysis, processingTime time.Duration) {
	ba.metrics.TotalAnalyses++
	ba.metrics.AnomaliesDetected += int64(len(analysis.Anomalies))
	ba.metrics.AverageConfidence = (ba.metrics.AverageConfidence + analysis.Confidence) / 2.0
	ba.metrics.ProcessingTime = processingTime
	ba.metrics.LastAnalysis = time.Now()
}

// GetMetrics returns the behavior analysis metrics
func (ba *BehaviorAnalyzer) GetMetrics() *BehaviorMetrics {
	return ba.metrics
}

// AddModel adds a machine learning model
func (ba *BehaviorAnalyzer) AddModel(name string, model interface{}, version string) {
	ba.modelsMutex.Lock()
	defer ba.modelsMutex.Unlock()

	ba.models[name] = &MLModel{
		Name:        name,
		Model:       model,
		Version:     version,
		LastUpdated: time.Now(),
		IsTrained:   false,
	}
}

// GetModel returns a machine learning model by name
func (ba *BehaviorAnalyzer) GetModel(name string) (*MLModel, bool) {
	ba.modelsMutex.RLock()
	defer ba.modelsMutex.RUnlock()

	model, exists := ba.models[name]
	return model, exists
}

// TrainModel trains a machine learning model
func (ba *BehaviorAnalyzer) TrainModel(name string, trainingData []*BehaviorData) error {
	model, exists := ba.GetModel(name)
	if !exists {
		return fmt.Errorf("model %s not found", name)
	}

	// In a real implementation, you would train the model with the provided data
	// For now, we'll just mark it as trained
	model.IsTrained = true
	model.Accuracy = 0.85 // Placeholder accuracy
	model.LastUpdated = time.Now()

	return nil
}
