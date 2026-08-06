export function isImageGenerationModel(model: string): boolean {
  const normalizedModel = model.toLowerCase()
  return (
    normalizedModel.includes('dall-e') ||
    normalizedModel.includes('gpt-image-') ||
    normalizedModel.includes('imagen-') ||
    normalizedModel.includes('flux-') ||
    normalizedModel.includes('flux.1-')
  )
}
