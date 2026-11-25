import baseballCapIcon from "@phosphor-icons/core/regular/baseball-cap.svg?raw"
import bracketsCurlyIcon from "@phosphor-icons/core/regular/brackets-curly.svg?raw"
import cameraIcon from "@phosphor-icons/core/regular/camera.svg?raw"
import circleDashedIcon from "@phosphor-icons/core/regular/circle-dashed.svg?raw"
import cloudIcon from "@phosphor-icons/core/regular/cloud.svg?raw"
import codeIcon from "@phosphor-icons/core/regular/code.svg?raw"
import crosshairIcon from "@phosphor-icons/core/regular/crosshair.svg?raw"
import cubeFocusIcon from "@phosphor-icons/core/regular/cube-focus.svg?raw"
import cubeTransparentIcon from "@phosphor-icons/core/regular/cube-transparent.svg?raw"
import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
import cursorClickIcon from "@phosphor-icons/core/regular/cursor-click.svg?raw"
import diamondsFourIcon from "@phosphor-icons/core/regular/diamonds-four.svg?raw"
import fileCodeIcon from "@phosphor-icons/core/regular/file-code.svg?raw"
import flameIcon from "@phosphor-icons/core/regular/flame.svg?raw"
import folderSimpleIcon from "@phosphor-icons/core/regular/folder-simple.svg?raw"
import frameCornersIcon from "@phosphor-icons/core/regular/frame-corners.svg?raw"
import gearSixIcon from "@phosphor-icons/core/regular/gear-six.svg?raw"
import graphIcon from "@phosphor-icons/core/regular/graph.svg?raw"
import handGrabbingIcon from "@phosphor-icons/core/regular/hand-grabbing.svg?raw"
import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
import intersectIcon from "@phosphor-icons/core/regular/intersect.svg?raw"
import lightbulbIcon from "@phosphor-icons/core/regular/lightbulb.svg?raw"
import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
import paintBucketIcon from "@phosphor-icons/core/regular/paint-bucket.svg?raw"
import personSimpleRunIcon from "@phosphor-icons/core/regular/person-simple-run.svg?raw"
import personIcon from "@phosphor-icons/core/regular/person.svg?raw"
import shapesIcon from "@phosphor-icons/core/regular/shapes.svg?raw"
import sparkleIcon from "@phosphor-icons/core/regular/sparkle.svg?raw"
import speakerHighIcon from "@phosphor-icons/core/regular/speaker-high.svg?raw"
import stackIcon from "@phosphor-icons/core/regular/stack.svg?raw"
import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
import userFocusIcon from "@phosphor-icons/core/regular/user-focus.svg?raw"
import wrenchIcon from "@phosphor-icons/core/regular/wrench.svg?raw"

const iconByClassName: Record<string, string> = {
	Model: shapesIcon,
	WorldModel: shapesIcon,
	Folder: folderSimpleIcon,
	Configuration: gearSixIcon,
	MeshPart: cubeTransparentIcon,
	UnionOperation: intersectIcon,
	NegateOperation: intersectIcon,
	IntersectOperation: intersectIcon,
	PartOperationAsset: intersectIcon,
	Accessory: baseballCapIcon,
	Accoutrement: baseballCapIcon,
	Hat: baseballCapIcon,
	Tool: wrenchIcon,
	HopperBin: wrenchIcon,
	Script: fileCodeIcon,
	LocalScript: codeIcon,
	ModuleScript: bracketsCurlyIcon,
	Attachment: crosshairIcon,
	Bone: graphIcon,
	Sound: speakerHighIcon,
	SoundGroup: speakerHighIcon,
	Decal: imageIcon,
	Texture: imageIcon,
	SurfaceAppearance: paintBucketIcon,
	BodyColors: paintBucketIcon,
	Shirt: tShirtIcon,
	Pants: tShirtIcon,
	ShirtGraphic: tShirtIcon,
	Humanoid: personIcon,
	HumanoidDescription: personIcon,
	Animation: personSimpleRunIcon,
	AnimationController: personSimpleRunIcon,
	Animator: personSimpleRunIcon,
	KeyframeSequence: personSimpleRunIcon,
	Keyframe: diamondsFourIcon,
	Pose: diamondsFourIcon,
	ParticleEmitter: sparkleIcon,
	Sparkles: sparkleIcon,
	Trail: sparkleIcon,
	Beam: sparkleIcon,
	Fire: flameIcon,
	Smoke: cloudIcon,
	Camera: cameraIcon,
	StringValue: textTIcon,
	TextLabel: textTIcon,
	TextButton: textTIcon,
	TextBox: textTIcon,
	ImageLabel: imageIcon,
	ImageButton: imageIcon,
	ClickDetector: cursorClickIcon,
	ProximityPrompt: handGrabbingIcon,
	WrapLayer: stackIcon,
	WrapTarget: stackIcon,
	FaceControls: userFocusIcon,
	Motor6D: linkIcon,
	Snap: linkIcon,
	Glue: linkIcon,
	Frame: frameCornersIcon,
	ScrollingFrame: frameCornersIcon,
}

// The icons of class families, by the end of their class names, such as
// PointLight and IntValue. The first match wins.
const iconByClassNameSuffix: [string, string][] = [
	["Script", fileCodeIcon],
	["Value", hashIcon],
	["Constraint", linkIcon],
	["Weld", linkIcon],
	["Light", lightbulbIcon],
	["Mesh", cubeFocusIcon],
	["Gui", frameCornersIcon],
	["Part", cubeIcon],
	["Seat", cubeIcon],
	["Location", cubeIcon],
]

export function classIcon(className: string): string {
	const icon = iconByClassName[className]
	if (icon) return icon
	for (const [suffix, suffixIcon] of iconByClassNameSuffix) {
		if (className.endsWith(suffix)) return suffixIcon
	}
	return circleDashedIcon
}
