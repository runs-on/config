package schema

// RepoConfig defines the structure of a runs-on.yml configuration file
#RepoConfig: {
	// Optional reference to another repository's config to extend
	_extends?: string

	// Map of runner specifications
	runners?: {
		[string]: #RunnerSpec
	}

	// Map of image specifications
	images?: {
		[string]: #ImageSpec
	}

	// Map of pool specifications
	pools?: {
		[string]: #PoolSpec
	}

	// List of admin usernames
	admins?: [...string]

	// Allow additional fields (for forward compatibility)
	...
}

// RunnerSpec defines a runner configuration
#RunnerSpec: {
	// Optional unique identifier for the runner
	id?: string

	// CPU count(s) - can be single int, string (e.g., "2+4"), or array
	cpu?: #IntArray

	// RAM in GB - can be single int, string (e.g., "16+32"), or array
	ram?: #IntArray

	// Disk size (DEPRECATED: use volume instead)
	disk?: string

	// Volume specification: format: size:type:throughput:iops
	// e.g., "80gb:gp3:125mbs:3000iops"
	volume?: string

	// Retry configuration - can be string (e.g., "always+on-failure") or array
	retry?: #StringArray

	// Extra features (e.g., "s3-cache", "efs") - can be string (e.g., "s3-cache+tmpfs") or array
	extras?: #StringArray

	// SSH access configuration (bool or string "true"/"false")
	ssh?: #BoolOrString

	// Private network configuration (bool or string "true"/"false")
	private?: #BoolOrString

	// Spot instance configuration
	// Values: "false", "never", "true", "pco", "price-capacity-optimized",
	//         "lp", "lowest-price", "co", "capacity-optimized"
	spot?: #SpotValue

	// Instance family - can be string (e.g., "c7a+m7a") or array (e.g., ["c7a", "m7a"])
	family?: #StringArray

	// Image reference
	image?: string

	// Preinstall script
	preinstall?: string

	// Tags for the runner
	tags?: #StringArray
}

// ImageSpec defines an image configuration
#ImageSpec: {
	// Optional unique identifier
	id?: string

	// Platform (e.g., "linux", "windows")
	platform?: string

	// Architecture (e.g., "x64", "arm64")
	arch?: string

	// Image name
	name?: string

	// Image owner
	owner?: string

	// Preinstall script
	preinstall?: string

	// AMI ID
	ami?: string

	// Main disk size in GB
	main_disk_size?: int & >=0

	// Root device name
	root_device_name?: string

	// Tags for the image
	tags?: {
		[string]: string
	}
}

// PoolSpec defines a pool configuration
#PoolSpec: {
	// Pool name (required, must match pattern)
	name: string & =~"^[a-z0-9_-]+$"

	// Pool version
	version?: string

	// Environment name (defaults to "production" if not set)
	env?: string

	// Timezone (defaults to "UTC" if not set)
	timezone?: string

	// Schedule configuration
	schedule?: [...#PoolSchedule]

	// Runner reference (required)
	runner: string

	// Maximum surge instances
	max_surge?: int & >=0
}

// PoolSchedule defines a schedule entry for a pool
#PoolSchedule: {
	// Schedule name
	name: string

	// Number of stopped instances
	stopped: int & >=0

	// Number of hot instances
	hot: int & >=0

	// Optional match criteria
	match?: #ScheduleMatch
}

// ScheduleMatch defines time-based matching criteria
#ScheduleMatch: {
	// Days of the week (e.g., ["monday", "tuesday"])
	day?: [...string]

	// Time ranges (e.g., ["22:00", "06:00"])
	time?: [...string]
}

// IntArray can be a single int, string representation, or array
// String values can use "+" separator (e.g., "2+4" is equivalent to [2, 4])
#IntArray: int | string | [...int] | [...string]

// StringArray can be a single string or array of strings
// String values can use "+" separator (e.g., "s3-cache+tmpfs" is equivalent to ["s3-cache", "tmpfs"])
#StringArray: string | [...string]

// BoolOrString can be a bool or string "true"/"false"
#BoolOrString: bool | "true" | "false"

// SpotValue defines valid spot instance configuration values
// Note: Boolean values (false/true) are automatically normalized to strings ("false"/"true") during validation
#SpotValue: "false" | "never" | "true" | "pco" | "price-capacity-optimized" | "lp" | "lowest-price" | "co" | "capacity-optimized"

// Main schema entry point
#Config: #RepoConfig

