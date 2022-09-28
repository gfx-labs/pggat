local jobs = [
    {name: "pggat", merge: {}},
];
local param_job(image,tag_var, merge = {}) = std.mergePatch({
    stage: 'build',
    image: {
      name: 'gcr.io/kaniko-project/executor:debug',
      entrypoint: [''],
    },
    script: [
        'mkdir -p /kaniko/.docker',
        @'echo "{\"auths\":{\"${CI_REGISTRY}\":{\"auth\":\"$(printf "%s:%s" "${CI_REGISTRY_USER}" "${CI_REGISTRY_PASSWORD}" | base64)\"}}}" > /kaniko/.docker/config.json',
        std.strReplace(|||
            /kaniko/executor
            --context ${CI_PROJECT_DIR}
            --cache=true
            --build-arg GOPROXY
            --cache-repo="${CI_REGISTRY_IMAGE}/kaniko/cache"
            --dockerfile "${CI_PROJECT_DIR}/Dockerfile"
            --destination "${CI_REGISTRY_IMAGE}/%(img)s:%(tag_var)s"
            --destination "${CI_REGISTRY_IMAGE}/%(img)s:latest"
        ||| % {img: image, tag_var: tag_var}, "\n", " "),
        ]
  }, merge);
{
  [job.name+"-tag"]: param_job(job.name,"${CI_COMMIT_TAG}",std.mergePatch(job.merge, {only:["tags"]}))
  for job in jobs
} + {
  [job.name]:  param_job(job.name,"${CI_COMMIT_SHORT_SHA}",job.merge)
  for job in jobs
}

