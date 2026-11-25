import {
	AttachmentWrapper,
	FindFirstMatchingAttachment,
	HumanoidDescriptionWrapper,
	RBXRenderer,
} from "roavatar-renderer"
import type { CFrame, Instance, Vector3 as RobloxVector3 } from "roavatar-renderer"
import {
	BufferGeometry,
	Group,
	Line,
	LineBasicMaterial,
	Quaternion,
	Vector3,
} from "three"
import type { AdjustmentAxis, AdjustmentKind } from "../avatar/avatar-draft"
import type { AdjustmentGuide } from "./model-subject"

// The guides of an accessory show the axes along which the adjustment moves,
// turns, and sizes it, as in the RoAvatar editor: lines through the handle
// for the position, rings around it for the rotation, and lines out of it for
// the scale. A guide is one stud long, scaled to the size of the handle.
interface AxisGuide {
	kind: AdjustmentKind
	axis: AdjustmentAxis
	// The part of the guide in front of the avatar.
	visibleMaterial: LineBasicMaterial
	// The part of the guide behind the avatar, which shows through it fainter.
	hiddenMaterial: LineBasicMaterial
}

const axes: AdjustmentAxis[] = ["x", "y", "z"],
	kinds: AdjustmentKind[] = ["position", "rotation", "scale"],
	ringPointCount = 64,
	// The opacity of each part of a guide, for the highlighted axis or for
	// every axis when none is highlighted, and for the other axes.
	visibleOpacity = { highlighted: 1, other: 0.3 },
	hiddenOpacity = { highlighted: 0.45, other: 0.12 },
	// The guides draw after the avatar, so that their hidden parts show
	// through it.
	guideRenderOrder = 1000

let guideGroup: Group | undefined,
	axisGuides: AxisGuide[] = []

// placeAdjustmentGuide shows the guides of the accessory in the guide on the
// rig, where the accessory sits now, or hides them without a guide.
export function placeAdjustmentGuide(
	guide: AdjustmentGuide | undefined,
	rig: Instance | undefined,
): void {
	const handlePlacement =
		guide && rig ? handlePlacementOf(rig, guide.assetId) : undefined
	if (!guide || !handlePlacement) {
		guideGroup?.removeFromParent()
		return
	}
	const group = guideGroupOf()
	if (!group.parent) RBXRenderer.scene.add(group)
	group.position.copy(handlePlacement.position)
	group.quaternion.copy(handlePlacement.orientation)
	group.scale.setScalar(handlePlacement.sizeStuds)
	for (const axisGuide of axisGuides) {
		axisGuide.visibleMaterial.visible = axisGuide.hiddenMaterial.visible =
			axisGuide.kind === guide.kind
		const isHighlighted = !guide.axis || axisGuide.axis === guide.axis
		axisGuide.visibleMaterial.opacity =
			visibleOpacity[isHighlighted ? "highlighted" : "other"]
		axisGuide.hiddenMaterial.opacity =
			hiddenOpacity[isHighlighted ? "highlighted" : "other"]
	}
}

export function removeAdjustmentGuides(): void {
	if (!guideGroup) return
	guideGroup.removeFromParent()
	guideGroup.traverse((child) => {
		if (child instanceof Line) child.geometry.dispose()
	})
	for (const axisGuide of axisGuides) {
		axisGuide.visibleMaterial.dispose()
		axisGuide.hiddenMaterial.dispose()
	}
	guideGroup = undefined
	axisGuides = []
}

function guideGroupOf(): Group {
	if (guideGroup) return guideGroup
	const group = new Group(),
		rootStyle = getComputedStyle(document.documentElement)
	for (const kind of kinds) {
		for (const axis of axes) {
			const color = rootStyle.getPropertyValue(`--color-axis-${axis}`).trim(),
				geometry = new BufferGeometry().setFromPoints(
					guidePointsOf(kind, axis),
				),
				visibleMaterial = new LineBasicMaterial({ color, transparent: true }),
				hiddenMaterial = new LineBasicMaterial({
					color,
					transparent: true,
					depthTest: false,
					depthWrite: false,
				}),
				visibleLine = new Line(geometry, visibleMaterial),
				hiddenLine = new Line(geometry, hiddenMaterial)
			visibleLine.renderOrder = hiddenLine.renderOrder = guideRenderOrder
			group.add(visibleLine, hiddenLine)
			axisGuides.push({ kind, axis, visibleMaterial, hiddenMaterial })
		}
	}
	guideGroup = group
	return group
}

function guidePointsOf(kind: AdjustmentKind, axis: AdjustmentAxis): Vector3[] {
	const direction = new Vector3().setComponent(axes.indexOf(axis), 1)
	if (kind === "position") return [direction.clone().negate(), direction]
	if (kind === "scale") return [new Vector3(), direction]
	// A ring turns about its axis, so it lies in the plane of the other two.
	const [firstAxisIndex, secondAxisIndex] = [0, 1, 2].filter(
		(axisIndex) => axisIndex !== axes.indexOf(axis),
	)
	return Array.from({ length: ringPointCount + 1 }, (_unused, pointIndex) => {
		const angle = (pointIndex / ringPointCount) * Math.PI * 2
		return new Vector3()
			.setComponent(firstAxisIndex, Math.cos(angle))
			.setComponent(secondAxisIndex, Math.sin(angle))
	})
}

// handlePlacementOf finds where the handle of the accessory sits. The guides
// turn with the attachment of the rig that holds the accessory, because
// Roblox applies the adjustment in the space of that attachment.
function handlePlacementOf(
	rig: Instance,
	assetId: number,
): { position: Vector3; orientation: Quaternion; sizeStuds: number } | undefined {
	const description = rig
			.FindFirstChildOfClass("Humanoid")
			?.FindFirstChildOfClass("HumanoidDescription"),
		accessoryDescription = description
			? new HumanoidDescriptionWrapper(description).getAccessoryDescriptionWithId(
					BigInt(assetId),
				)
			: undefined,
		accessory = accessoryDescription?.Prop("Instance") as Instance | undefined,
		handle = accessory?.FindFirstChild("Handle")
	if (!handle?.className.includes("Part")) return undefined
	const placement = (handle.Prop("CFrame") as CFrame).clone(),
		handleSize = handle.Prop("Size") as RobloxVector3
	placement.Orientation = [0, 0, 0]
	for (const attachment of handle.GetChildren()) {
		if (attachment.className !== "Attachment") continue
		const rigAttachment = FindFirstMatchingAttachment(
			attachment.Prop("Name") as string,
			rig,
		)
		if (!rigAttachment) continue
		placement.Orientation = new AttachmentWrapper(
			rigAttachment,
		).getWorldCFrame().Orientation
		break
	}
	const position = new Vector3(),
		orientation = new Quaternion()
	placement.getTHREEMatrix().decompose(position, orientation, new Vector3())
	return {
		position,
		orientation,
		sizeStuds: Math.max(1, (handleSize.X + handleSize.Y + handleSize.Z) / 3),
	}
}
