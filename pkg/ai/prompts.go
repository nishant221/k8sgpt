package ai

const (
	default_prompt = `Simplify the following Kubernetes error message delimited by triple dashes written in --- %s --- language; --- %s ---.
	Provide the most possible solution in a step by step style in no more than 10000 characters. Write the output in the following format:
	Error: {Explain error here}
	Solution: {Step by step solution here}
	`
)
