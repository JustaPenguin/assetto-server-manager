let gulp = require('gulp');
let browserify = require('browserify');
let source = require('vinyl-source-stream');
let tsify = require('tsify');
let sourcemaps = require('gulp-sourcemaps');
let buffer = require('vinyl-buffer');
let uglify = require('gulp-uglify');

gulp.task('build', buildJS);

gulp.task('watch', () => {
    return gulp.watch(['src/**/*.ts', 'src/**/*.js'], buildJS);
});

gulp.task('default', gulp.series('build', 'watch'));

function buildJS() {
    try {
        return browserify({
            basedir: '.',
            debug: false,
            entries: ['src/main.ts'],
            cache: {},
            packageCache: {},
        })
            .plugin(tsify)
            .transform('babelify', {
                presets: ['@babel/preset-env'],
                extensions: ['.ts']
            })
            .transform({global: true}, 'browserify-shim')
            .bundle()
            .pipe(source('bundle.js'))
            .pipe(buffer())
            .pipe(sourcemaps.init({loadMaps: true}))
            .pipe(uglify())
            .pipe(sourcemaps.write('../static/js'))
            .pipe(gulp.dest('../static/js'));

    } catch (e) {
        console.error(e);
    }
}
