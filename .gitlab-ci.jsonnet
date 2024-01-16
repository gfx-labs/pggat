local jobs = [
  { name: 'pggat', merge: {} },
];
local param_job(image, tag_var, merge={}) = std.mergePatch({
  stage: 'build',
  image: {
    name: 'gcr.io/kaniko-project/executor:debug',
    entrypoint: [''],
  },
  script: [
    'mkdir -p /kaniko/.docker',
    @'echo "{\"auths\":{\"${HARBOR_HOST}\":{\"auth\":\"$(echo -n ${HARBOR_USERNAME}:${HARBOR_PASSWORD} | base64)\"}}}" > /kaniko/.docker/config.json',
    std.strReplace(|||
      /kaniko/executor
      --context ${CI_PROJECT_DIR}
      --cache=true
      --cache-repo="${HARBOR_HOST}/kaniko/cache"
      --compressed-caching=false
      --build-arg GOPROXY
      --registry-mirror=mirror.gfx.cafe
      --registry-mirror=mirror.gcr.io
      --registry-mirror=index.docker.io
      --dockerfile "${CI_PROJECT_DIR}/%(img)s.Dockerfile"
      --destination "${HARBOR_HOST}/${HARBOR_PROJECT}/${CI_PROJECT_PATH}/%(img)s:%(tag_var)s"
      --destination "${HARBOR_HOST}/${HARBOR_PROJECT}/${CI_PROJECT_PATH}/%(img)s:${CI_COMMIT_SHORT_SHA}"
      --destination "${HARBOR_HOST}/${HARBOR_PROJECT}/${CI_PROJECT_PATH}/%(img)s:latest"
      --snapshotMode=redo
    ||| % { img: image, tag_var: tag_var }, '\n', ' '),
  ],
}, merge);
{
  [job.name + '-tag']: param_job(job.name, '${CI_COMMIT_TAG}', std.mergePatch(job.merge, { only: ['tags'] }))
  for job in jobs
} + {
  [job.name]: param_job(job.name, '${CI_COMMIT_BRANCH}', job.merge)
  for job in jobs
}
